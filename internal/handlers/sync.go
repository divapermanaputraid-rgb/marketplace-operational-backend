package handlers

import (
	"net/http"
	"time"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
)

type SyncHandler struct {
	syncRepo  *repositories.SyncRepository
	storeRepo *repositories.StoreRepository
}

func NewSyncHandler(syncRepo *repositories.SyncRepository, storeRepo *repositories.StoreRepository) *SyncHandler {
	return &SyncHandler{
		syncRepo:  syncRepo,
		storeRepo: storeRepo,
	}
}

type CreateSyncJobRequest struct {
	StoreID                 *string `json:"store_id"`
	Marketplace             string  `json:"marketplace" binding:"required"`
	SyncType                string  `json:"sync_type" binding:"required"`
	SyncDirection           string  `json:"sync_direction" binding:"required"`
	JobName                 string  `json:"job_name" binding:"required"`
	IsActive                *bool   `json:"is_active"`
	ScheduleEnabled         *bool   `json:"schedule_enabled"`
	ScheduleIntervalMinutes *int    `json:"schedule_interval_minutes"`
}

type UpdateSyncJobRequest struct {
	JobName                 *string `json:"job_name"`
	Status                  *string `json:"status"`
	IsActive                *bool   `json:"is_active"`
	ScheduleEnabled         *bool   `json:"schedule_enabled"`
	ScheduleIntervalMinutes *int    `json:"schedule_interval_minutes"`
}

func (h *SyncHandler) ListJobs(c *gin.Context) {
	storeID := c.Query("store_id")
	marketplace := c.Query("marketplace")
	syncType := c.Query("sync_type")
	status := c.Query("status")

	jobs, err := h.syncRepo.ListSyncJobs(storeID, marketplace, syncType, status)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync jobs"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(jobs, ""))
}

func (h *SyncHandler) GetJob(c *gin.Context) {
	id := c.Param("id")
	job, err := h.syncRepo.GetSyncJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, ""))
}

func (h *SyncHandler) CreateJob(c *gin.Context) {
	var req CreateSyncJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", err.Error()))
		return
	}

	var storeIDPtr *uuid.UUID
	if req.StoreID != nil && *req.StoreID != "" {
		id, err := uuid.Parse(*req.StoreID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid store ID format"))
			return
		}
		storeIDPtr = &id

		// Validate Store
		store, err := h.storeRepo.FindByID(*req.StoreID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Store not found"))
			return
		}

		if req.Marketplace != "all" && store.Marketplace != req.Marketplace {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Marketplace must match store marketplace or be 'all'"))
			return
		}
	}

	isActive := true
	if req.IsActive != nil {
		isActive = *req.IsActive
	}

	scheduleEnabled := false
	if req.ScheduleEnabled != nil {
		scheduleEnabled = *req.ScheduleEnabled
	}

	job := &models.SyncJob{
		StoreID:                 storeIDPtr,
		Marketplace:             req.Marketplace,
		SyncType:                req.SyncType,
		SyncDirection:           req.SyncDirection,
		JobName:                 req.JobName,
		Status:                  "idle",
		IsActive:                isActive,
		ScheduleEnabled:         scheduleEnabled,
		ScheduleIntervalMinutes: req.ScheduleIntervalMinutes,
	}

	if err := h.syncRepo.CreateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to create sync job"))
		return
	}

	c.JSON(http.StatusCreated, models.SuccessResponse(job, "Sync job created successfully"))
}

func (h *SyncHandler) UpdateJob(c *gin.Context) {
	id := c.Param("id")
	var req UpdateSyncJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("VALIDATION_ERROR", "Invalid input data"))
		return
	}

	job, err := h.syncRepo.GetSyncJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	if req.JobName != nil {
		job.JobName = *req.JobName
	}
	if req.Status != nil {
		job.Status = *req.Status
	}
	if req.IsActive != nil {
		job.IsActive = *req.IsActive
	}
	if req.ScheduleEnabled != nil {
		job.ScheduleEnabled = *req.ScheduleEnabled
	}
	if req.ScheduleIntervalMinutes != nil {
		job.ScheduleIntervalMinutes = req.ScheduleIntervalMinutes
	}

	if err := h.syncRepo.UpdateSyncJob(job); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to update sync job"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, "Sync job updated successfully"))
}

func (h *SyncHandler) DeleteJob(c *gin.Context) {
	id := c.Param("id")
	if err := h.syncRepo.SoftDeleteSyncJob(id); err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to delete sync job"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(nil, "Sync job deleted successfully"))
}

func (h *SyncHandler) RunJob(c *gin.Context) {
	id := c.Param("id")

	job, err := h.syncRepo.GetSyncJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", err.Error()))
		return
	}

	// Update job status to running initially (simulating real run)
	now := time.Now()
	job.LastRunAt = &now
	job.Status = "running"
	_ = h.syncRepo.UpdateSyncJob(job)

	// In Sprint 8, we do not have real API integrations.
	// We simulate a skipped/not_configured run immediately.
	statusMsg := "Marketplace API integration is not configured yet. Sync was skipped."
	finalStatus := "not_configured"

	// Create SyncLog
	durationMs := int64(time.Since(now).Milliseconds())
	log := &models.SyncLog{
		SyncJobID:     &job.ID,
		StoreID:       job.StoreID,
		Marketplace:   job.Marketplace,
		SyncType:      job.SyncType,
		SyncDirection: job.SyncDirection,
		Status:        finalStatus,
		Message:       &statusMsg,
		StartedAt:     &now,
		FinishedAt:    &now,
		DurationMs:    &durationMs,
	}

	_ = h.syncRepo.CreateSyncLog(log)

	// Finalize Job status
	job.Status = finalStatus
	_ = h.syncRepo.UpdateSyncJob(job)

	// Re-fetch job to include related data if any
	updatedJob, _ := h.syncRepo.GetSyncJobByID(id)

	c.JSON(http.StatusOK, models.SuccessResponse(updatedJob, statusMsg))
}

func (h *SyncHandler) ListLogs(c *gin.Context) {
	jobID := c.Query("sync_job_id")
	storeID := c.Query("store_id")
	marketplace := c.Query("marketplace")
	syncType := c.Query("sync_type")
	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	logs, err := h.syncRepo.ListSyncLogs(jobID, storeID, marketplace, syncType, status, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync logs"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(logs, ""))
}

func (h *SyncHandler) ListJobLogs(c *gin.Context) {
	id := c.Param("id")
	logs, err := h.syncRepo.ListSyncLogs(id, "", "", "", "", "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync logs"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(logs, ""))
}
