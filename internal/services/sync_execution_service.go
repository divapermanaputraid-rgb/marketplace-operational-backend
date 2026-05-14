package services

import (
	"errors"
	"fmt"
	"time"

	"github.com/marketplace-ops/backend/internal/marketplace"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/security"
)

type SyncExecutionService struct {
	syncRepo        *repositories.SyncRepository
	storeRepo       *repositories.StoreRepository
	integrationRepo *repositories.IntegrationRepository
	orderRepo       *repositories.OrderRepository
	mappingRepo     *repositories.ProductMappingRepository
	productRepo     *repositories.ProductRepository
}

func NewSyncExecutionService(
	syncRepo *repositories.SyncRepository,
	storeRepo *repositories.StoreRepository,
	integrationRepo *repositories.IntegrationRepository,
	orderRepo *repositories.OrderRepository,
	mappingRepo *repositories.ProductMappingRepository,
	productRepo *repositories.ProductRepository,
) *SyncExecutionService {
	return &SyncExecutionService{
		syncRepo:        syncRepo,
		storeRepo:       storeRepo,
		integrationRepo: integrationRepo,
		orderRepo:       orderRepo,
		mappingRepo:     mappingRepo,
		productRepo:     productRepo,
	}
}

func (s *SyncExecutionService) ExecuteJob(job *models.SyncJob) (*models.SyncLog, error) {
	startTime := time.Now()

	// Update job status to running
	job.Status = "running"
	job.LastRunAt = &startTime
	_ = s.syncRepo.UpdateSyncJob(job)

	syncLog := &models.SyncLog{
		SyncJobID:     &job.ID,
		StoreID:       job.StoreID,
		Marketplace:   job.Marketplace,
		SyncType:      job.SyncType,
		SyncDirection: job.SyncDirection,
		StartedAt:     &startTime,
		Status:        "started",
	}

	var err error
	switch job.SyncType {
	case "orders":
		if job.SyncDirection == "pull" {
			err = s.executePullOrders(job, syncLog)
		} else {
			err = errors.New("unsupported sync direction for orders")
		}
	case "products":
		if job.SyncDirection == "pull" {
			err = s.executePullProducts(job, syncLog)
		} else {
			err = errors.New("unsupported sync direction for products")
		}
	case "stock":
		if job.SyncDirection == "push" {
			// For foundation, we don't auto-push all stock yet.
			// Only manual push exists via another endpoint.
			statusMsg := "Automatic bulk stock push is not enabled yet for safety."
			syncLog.Status = "skipped"
			syncLog.Message = &statusMsg
		} else {
			err = errors.New("unsupported sync direction for stock")
		}
	default:
		err = fmt.Errorf("unsupported sync type: %s", job.SyncType)
	}

	endTime := time.Now()
	durationMs := int64(endTime.Sub(startTime).Milliseconds())
	syncLog.FinishedAt = &endTime
	syncLog.DurationMs = &durationMs

	if err != nil {
		errMsg := err.Error()
		syncLog.Status = "failed"
		syncLog.ErrorMessage = &errMsg
		if syncLog.Message == nil {
			syncLog.Message = &errMsg
		}
	}

	// Save log
	_ = s.syncRepo.CreateSyncLog(syncLog)

	// Update job final status
	job.Status = syncLog.Status
	if syncLog.Status == "success" || syncLog.Status == "partial" {
		job.LastSuccessAt = &endTime
	}
	if syncLog.ErrorMessage != nil {
		job.LastError = syncLog.ErrorMessage
	}
	_ = s.syncRepo.UpdateSyncJob(job)

	return syncLog, err
}

func (s *SyncExecutionService) executePullOrders(job *models.SyncJob, syncLog *models.SyncLog) error {
	if job.StoreID == nil {
		return errors.New("store ID is required for order pull")
	}

	store, err := s.storeRepo.FindByID(job.StoreID.String())
	if err != nil {
		return err
	}

	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		return err
	}

	cred, _ := s.integrationRepo.FindCredentialByStoreAndMarketplace(store.ID.String(), store.Marketplace)
	if cred == nil {
		syncLog.Status = "not_configured"
		msg := "No credentials configured for this marketplace."
		syncLog.Message = &msg
		return nil
	}

	// Validate token expiration
	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		syncLog.Status = "expired"
		msg := "Access token expired. Please reconnect."
		syncLog.Message = &msg
		return nil
	}

	if cred.EncryptedAccessToken == nil || *cred.EncryptedAccessToken == "" {
		return errors.New("access token is missing")
	}

	accessToken, err := security.DecryptToken(*cred.EncryptedAccessToken)
	if err != nil {
		return err
	}

	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	// For automated pull, we use a default window (e.g., last 24 hours)
	timeTo := time.Now().Unix()
	timeFrom := timeTo - (24 * 3600)

	listRes, err := adapter.PullOrders(accessToken, extStoreID, timeFrom, timeTo, 50, "")
	if err != nil {
		return err
	}

	var orderSNs []string
	for _, o := range listRes.Orders {
		orderSNs = append(orderSNs, o.OrderSN)
	}

	var details []marketplace.ShopeeOrderDetail
	if len(orderSNs) > 0 {
		details, err = adapter.GetOrderDetails(accessToken, extStoreID, orderSNs)
		if err != nil {
			return err
		}
	}

	mapper := NewShopeeOrderMapper(s.orderRepo, s.mappingRepo)
	var recordsCreated, recordsUpdated, recordsFailed int

	for _, detail := range details {
		created, updated, _, mapErr := mapper.MapAndPersist(store.ID, detail)
		if mapErr != nil {
			recordsFailed++
			continue
		}
		if created {
			recordsCreated++
		} else if updated {
			recordsUpdated++
		}
	}

	syncLog.RecordsProcessed = len(details)
	syncLog.RecordsCreated = recordsCreated
	syncLog.RecordsUpdated = recordsUpdated
	syncLog.RecordsFailed = recordsFailed

	if recordsFailed > 0 {
		if recordsCreated > 0 || recordsUpdated > 0 {
			syncLog.Status = "partial"
		} else {
			syncLog.Status = "failed"
		}
	} else {
		syncLog.Status = "success"
	}

	return nil
}

func (s *SyncExecutionService) executePullProducts(job *models.SyncJob, syncLog *models.SyncLog) error {
	// Product pull foundation for Shopee
	if job.StoreID == nil {
		return errors.New("store ID is required for product pull")
	}

	store, err := s.storeRepo.FindByID(job.StoreID.String())
	if err != nil {
		return err
	}

	adapter, err := marketplace.GetAdapter(store.Marketplace)
	if err != nil {
		return err
	}

	cred, _ := s.integrationRepo.FindCredentialByStoreAndMarketplace(store.ID.String(), store.Marketplace)
	if cred == nil {
		syncLog.Status = "not_configured"
		msg := "No credentials configured."
		syncLog.Message = &msg
		return nil
	}

	accessToken, err := security.DecryptToken(*cred.EncryptedAccessToken)
	if err != nil {
		return err
	}

	extStoreID := ""
	if store.ExternalStoreID != nil {
		extStoreID = *store.ExternalStoreID
	}

	// Simple pull for foundation
	res, err := adapter.PullProducts(accessToken, extStoreID, 0, 50, "NORMAL")
	if err != nil {
		return err
	}

	syncLog.RecordsProcessed = len(res.Items)
	syncLog.Status = "success"
	msg := fmt.Sprintf("Successfully pulled %d products from marketplace.", len(res.Items))
	syncLog.Message = &msg

	return nil
}
