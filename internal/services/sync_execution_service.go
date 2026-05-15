package services

import (
	"encoding/json"
	"errors"
	"fmt"
	"strconv"
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
	inventoryRepo   *repositories.InventoryRepository
}

func NewSyncExecutionService(
	syncRepo *repositories.SyncRepository,
	storeRepo *repositories.StoreRepository,
	integrationRepo *repositories.IntegrationRepository,
	orderRepo *repositories.OrderRepository,
	mappingRepo *repositories.ProductMappingRepository,
	productRepo *repositories.ProductRepository,
	inventoryRepo *repositories.InventoryRepository,
) *SyncExecutionService {
	return &SyncExecutionService{
		syncRepo:        syncRepo,
		storeRepo:       storeRepo,
		integrationRepo: integrationRepo,
		orderRepo:       orderRepo,
		mappingRepo:     mappingRepo,
		productRepo:     productRepo,
		inventoryRepo:   inventoryRepo,
	}
}

type SyncJobConfig struct {
	LookbackMinutes  int    `json:"lookback_minutes,omitempty"`
	PageSize         int    `json:"page_size,omitempty"`
	OrderStatus      string `json:"order_status,omitempty"`
	ItemStatus       string `json:"item_status,omitempty"`
	ProductMappingID string `json:"product_mapping_id,omitempty"`
	DryRun           bool   `json:"dry_run,omitempty"`
}

func (s *SyncExecutionService) ExecuteJob(job *models.SyncJob) (*models.SyncLog, error) {
	startTime := time.Now()

	if !job.IsActive {
		return nil, errors.New("job is inactive")
	}

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

	// Parse config
	var config SyncJobConfig
	if job.Config != nil && *job.Config != "" {
		_ = json.Unmarshal([]byte(*job.Config), &config)
	}

	var err error
	switch job.SyncType {
	case "orders":
		if job.SyncDirection == "pull" {
			err = s.executePullOrders(job, &config, syncLog)
		} else {
			err = errors.New("unsupported sync direction for orders")
		}
	case "products":
		if job.SyncDirection == "pull" {
			err = s.executePullProducts(job, &config, syncLog)
		} else {
			err = errors.New("unsupported sync direction for products")
		}
	case "stock":
		if job.SyncDirection == "push" {
			err = s.executePushStock(job, &config, syncLog)
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

func (s *SyncExecutionService) executePullOrders(job *models.SyncJob, config *SyncJobConfig, syncLog *models.SyncLog) error {
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

	// Use config for window
	lookback := 24 * 60 // Default 24h
	if config.LookbackMinutes > 0 {
		lookback = config.LookbackMinutes
	}
	timeTo := time.Now().Unix()
	timeFrom := timeTo - int64(lookback*60)

	pageSize := 50
	if config.PageSize > 0 && config.PageSize <= 100 {
		pageSize = config.PageSize
	}

	listRes, err := adapter.PullOrders(accessToken, extStoreID, timeFrom, timeTo, pageSize, "")
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
	var pullErrors []string

	for _, detail := range details {
		created, updated, _, mapErr := mapper.MapAndPersist(store.ID, detail)
		if mapErr != nil {
			recordsFailed++
			pullErrors = append(pullErrors, fmt.Sprintf("Order %s: %v", detail.OrderSN, mapErr))
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

	summary := map[string]interface{}{
		"time_from":         timeFrom,
		"time_to":           timeTo,
		"lookback_minutes":  lookback,
		"page_size":         pageSize,
		"order_status":      config.OrderStatus,
		"records_processed": len(details),
		"records_created":   recordsCreated,
		"records_updated":   recordsUpdated,
		"records_failed":    recordsFailed,
	}
	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)
	syncLog.RawSummary = &summaryStr

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

func (s *SyncExecutionService) executePullProducts(job *models.SyncJob, config *SyncJobConfig, syncLog *models.SyncLog) error {
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

	// Validate token expiration
	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		syncLog.Status = "expired"
		msg := "Access token expired."
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

	pageSize := 50
	if config.PageSize > 0 && config.PageSize <= 50 {
		pageSize = config.PageSize
	}

	itemStatus := "NORMAL"
	if config.ItemStatus != "" {
		itemStatus = config.ItemStatus
	}

	res, err := adapter.PullProducts(accessToken, extStoreID, 0, pageSize, itemStatus)
	if err != nil {
		return err
	}

	syncLog.RecordsProcessed = len(res.Items)
	syncLog.Status = "success"
	msg := fmt.Sprintf("Successfully pulled %d product IDs from marketplace.", len(res.Items))
	syncLog.Message = &msg

	summary := map[string]interface{}{
		"page_size":         pageSize,
		"item_status":       itemStatus,
		"records_processed": len(res.Items),
		"has_next_page":     res.HasNextPage,
		"next_offset":       res.NextOffset,
	}
	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)
	syncLog.RawSummary = &summaryStr

	return nil
}

func (s *SyncExecutionService) executePushStock(job *models.SyncJob, config *SyncJobConfig, syncLog *models.SyncLog) error {
	if job.StoreID == nil {
		return errors.New("store ID is required for stock push")
	}

	// Mandatory: product_mapping_id
	if config.ProductMappingID == "" {
		syncLog.Status = "skipped"
		msg := "Skipped: Automated bulk stock push is disabled. Explicit product_mapping_id is required in job config."
		syncLog.Message = &msg
		return nil
	}

	store, err := s.storeRepo.FindByID(job.StoreID.String())
	if err != nil {
		return err
	}

	// 1. Get mapping
	mapping, err := s.mappingRepo.GetByID(config.ProductMappingID)
	if err != nil {
		return fmt.Errorf("mapping not found: %s", config.ProductMappingID)
	}

	if mapping.StoreID.String() != store.ID.String() {
		return errors.New("mapping does not belong to this store")
	}

	if mapping.ExternalProductID == "" {
		return errors.New("mapping missing external product ID")
	}

	// 2. Get inventory item
	var inventoryItem *models.InventoryItem
	if mapping.ProductVariantID != nil && mapping.ProductVariantID.String() != "" {
		inventoryItem, err = s.inventoryRepo.GetByProductAndVariant(mapping.ProductID.String(), mapping.ProductVariantID.String(), "Main Warehouse")
	} else {
		inventoryItem, err = s.inventoryRepo.GetByProductAndVariant(mapping.ProductID.String(), "", "Main Warehouse")
	}

	if err != nil || inventoryItem == nil {
		return errors.New("internal inventory item not found")
	}

	quantity := inventoryItem.AvailableQuantity

	summary := ginH{
		"product_mapping_id":  mapping.ID.String(),
		"external_product_id": mapping.ExternalProductID,
		"pushed_quantity":     quantity,
		"dry_run":             config.DryRun,
	}
	if mapping.ExternalVariantID != nil {
		summary["external_variant_id"] = *mapping.ExternalVariantID
	}

	if config.DryRun {
		syncLog.Status = "dry_run"
		msg := fmt.Sprintf("Dry run: Would push %d to item %s", quantity, mapping.ExternalProductID)
		syncLog.Message = &msg
		syncLog.RecordsProcessed = 1

		summaryJSON, _ := json.Marshal(summary)
		summaryStr := string(summaryJSON)
		syncLog.RawSummary = &summaryStr
		return nil
	}

	// 3. Real push
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

	if cred.AccessTokenExpiresAt != nil && time.Now().After(*cred.AccessTokenExpiresAt) {
		syncLog.Status = "expired"
		msg := "Access token expired."
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

	extProdIDInt, _ := strconv.ParseInt(mapping.ExternalProductID, 10, 64)
	extVarIDInt := int64(0)
	if mapping.ExternalVariantID != nil && *mapping.ExternalVariantID != "" {
		extVarIDInt, _ = strconv.ParseInt(*mapping.ExternalVariantID, 10, 64)
	}

	err = adapter.UpdateStock(accessToken, extStoreID, extProdIDInt, extVarIDInt, quantity)
	if err != nil {
		return err
	}

	syncLog.Status = "success"
	syncLog.RecordsProcessed = 1
	msg := fmt.Sprintf("Successfully pushed %d to item %s", quantity, mapping.ExternalProductID)
	syncLog.Message = &msg

	summaryJSON, _ := json.Marshal(summary)
	summaryStr := string(summaryJSON)
	syncLog.RawSummary = &summaryStr

	return nil
}

type ginH map[string]interface{}
