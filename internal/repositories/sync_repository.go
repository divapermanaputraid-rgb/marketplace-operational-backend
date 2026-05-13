package repositories

import (
	"errors"

	"github.com/marketplace-ops/backend/internal/models"
	"gorm.io/gorm"
)

type SyncRepository struct {
	db *gorm.DB
}

func NewSyncRepository(db *gorm.DB) *SyncRepository {
	return &SyncRepository{db: db}
}

func (r *SyncRepository) ListSyncJobs(storeID, marketplace, syncType, status string) ([]models.SyncJob, error) {
	var jobs []models.SyncJob
	query := r.db.Model(&models.SyncJob{}).Preload("Store")

	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if marketplace != "" {
		query = query.Where("marketplace = ?", marketplace)
	}
	if syncType != "" {
		query = query.Where("sync_type = ?", syncType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}

	err := query.Order("created_at desc").Find(&jobs).Error
	return jobs, err
}

func (r *SyncRepository) GetSyncJobByID(id string) (*models.SyncJob, error) {
	var job models.SyncJob
	err := r.db.Preload("Store").Where("id = ?", id).First(&job).Error
	if err != nil {
		if errors.Is(err, gorm.ErrRecordNotFound) {
			return nil, errors.New("sync job not found")
		}
		return nil, err
	}
	return &job, nil
}

func (r *SyncRepository) CreateSyncJob(job *models.SyncJob) error {
	return r.db.Create(job).Error
}

func (r *SyncRepository) UpdateSyncJob(job *models.SyncJob) error {
	return r.db.Save(job).Error
}

func (r *SyncRepository) SoftDeleteSyncJob(id string) error {
	return r.db.Delete(&models.SyncJob{}, "id = ?", id).Error
}

func (r *SyncRepository) ListSyncLogs(jobID, storeID, marketplace, syncType, status, dateFrom, dateTo string) ([]models.SyncLog, error) {
	var logs []models.SyncLog
	query := r.db.Model(&models.SyncLog{}).Preload("Store").Preload("SyncJob")

	if jobID != "" {
		query = query.Where("sync_job_id = ?", jobID)
	}
	if storeID != "" {
		query = query.Where("store_id = ?", storeID)
	}
	if marketplace != "" {
		query = query.Where("marketplace = ?", marketplace)
	}
	if syncType != "" {
		query = query.Where("sync_type = ?", syncType)
	}
	if status != "" {
		query = query.Where("status = ?", status)
	}
	if dateFrom != "" {
		query = query.Where("created_at >= ?", dateFrom)
	}
	if dateTo != "" {
		query = query.Where("created_at <= ?", dateTo)
	}

	err := query.Order("created_at desc").Find(&logs).Error
	return logs, err
}

func (r *SyncRepository) CreateSyncLog(log *models.SyncLog) error {
	return r.db.Create(log).Error
}
