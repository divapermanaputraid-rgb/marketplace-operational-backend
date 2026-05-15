package handlers

import (
	"net/http"

	"github.com/gin-gonic/gin"
	"github.com/google/uuid"
	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/services"
)

type SyncHandler struct {
	syncRepo    *repositories.SyncRepository
	storeRepo   *repositories.StoreRepository
	syncService *services.SyncExecutionService
}

func NewSyncHandler(
	syncRepo *repositories.SyncRepository,
	storeRepo *repositories.StoreRepository,
	syncService *services.SyncExecutionService,
) *SyncHandler {
	return &SyncHandler{
		syncRepo:    syncRepo,
		storeRepo:   storeRepo,
		syncService: syncService,
	}
}

type CreateSyncJobRequest struct {
	StoreID                 *string `json:"store_id"`
	Marketplace             string  `json:"marketplace" binding:"required"`
	SyncType                string  `json:"sync_type" binding:"required"`
	SyncDirection           string  `json:"sync_direction" binding:"required"`
	JobName                 string  `json:"job_name" binding:"required"`
	IsActive                bool    `json:"is_active"`
	ScheduleEnabled         bool    `json:"schedule_enabled"`
	ScheduleIntervalMinutes *int    `json:"schedule_interval_minutes"`
	Config                  *string `json:"config"`
}

type UpdateSyncJobRequest struct {
	JobName                 *string `json:"job_name"`
	IsActive                *bool   `json:"is_active"`
	ScheduleEnabled         *bool   `json:"schedule_enabled"`
	ScheduleIntervalMinutes *int    `json:"schedule_interval_minutes"`
	Config                  *string `json:"config"`
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
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Sync job not found"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(job, ""))
}

func (h *SyncHandler) CreateJob(c *gin.Context) {
	var req CreateSyncJobRequest
	if err := c.ShouldBindJSON(&req); err != nil {
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	job := &models.SyncJob{
		Marketplace:             req.Marketplace,
		SyncType:                req.SyncType,
		SyncDirection:           req.SyncDirection,
		JobName:                 req.JobName,
		IsActive:                req.IsActive,
		ScheduleEnabled:         req.ScheduleEnabled,
		ScheduleIntervalMinutes: req.ScheduleIntervalMinutes,
		Status:                  "idle",
		Config:                  req.Config,
	}

	if req.StoreID != nil && *req.StoreID != "" {
		storeUUID, err := uuid.Parse(*req.StoreID)
		if err != nil {
			c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_STORE_ID", "Invalid store ID format"))
			return
		}
		job.StoreID = &storeUUID
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
		c.JSON(http.StatusBadRequest, models.ErrorResponse("INVALID_REQUEST", err.Error()))
		return
	}

	job, err := h.syncRepo.GetSyncJobByID(id)
	if err != nil {
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Sync job not found"))
		return
	}

	if req.JobName != nil {
		job.JobName = *req.JobName
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
	if req.Config != nil {
		job.Config = req.Config
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
		c.JSON(http.StatusNotFound, models.ErrorResponse("NOT_FOUND", "Sync job not found"))
		return
	}

	// For manual run, we just execute it synchronously in the handler for feedback
	syncLog, err := h.syncService.ExecuteJob(job)

	response := gin.H{
		"status":            "failed",
		"message":           "",
		"sync_log_id":       "",
		"records_processed": 0,
		"records_created":   0,
		"records_updated":   0,
		"records_failed":    0,
		"errors":            []string{},
		"started_at":        nil,
		"finished_at":       nil,
	}

	if syncLog != nil {
		response["sync_log_id"] = syncLog.ID
		response["status"] = syncLog.Status
		response["records_processed"] = syncLog.RecordsProcessed
		response["records_created"] = syncLog.RecordsCreated
		response["records_updated"] = syncLog.RecordsUpdated
		response["records_failed"] = syncLog.RecordsFailed
		response["started_at"] = syncLog.StartedAt
		response["finished_at"] = syncLog.FinishedAt
		if syncLog.Message != nil {
			response["message"] = *syncLog.Message
		}
		if syncLog.ErrorMessage != nil {
			response["errors"] = []string{*syncLog.ErrorMessage}
		}
	}

	if err != nil {
		response["message"] = "Job completed with error: " + err.Error()
		c.JSON(http.StatusOK, models.SuccessResponse(response, response["message"].(string)))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(response, "Sync job executed successfully"))
}

func (h *SyncHandler) ListLogs(c *gin.Context) {
	storeID := c.Query("store_id")
	marketplace := c.Query("marketplace")
	syncType := c.Query("sync_type")
	status := c.Query("status")
	dateFrom := c.Query("date_from")
	dateTo := c.Query("date_to")

	logs, err := h.syncRepo.ListSyncLogs("", storeID, marketplace, syncType, status, dateFrom, dateTo)
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch sync logs"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(logs, ""))
}

func (h *SyncHandler) ListJobLogs(c *gin.Context) {
	jobID := c.Param("id")
	status := c.Query("status")

	logs, err := h.syncRepo.ListSyncLogs(jobID, "", "", "", status, "", "")
	if err != nil {
		c.JSON(http.StatusInternalServerError, models.ErrorResponse("INTERNAL_ERROR", "Failed to fetch job logs"))
		return
	}

	c.JSON(http.StatusOK, models.SuccessResponse(logs, ""))
}
