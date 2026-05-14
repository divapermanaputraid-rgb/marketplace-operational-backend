package workers

import (
	"context"
	"log"
	"time"

	"github.com/marketplace-ops/backend/internal/models"
	"github.com/marketplace-ops/backend/internal/repositories"
	"github.com/marketplace-ops/backend/internal/services"
)

type SyncWorker struct {
	syncRepo    *repositories.SyncRepository
	syncService *services.SyncExecutionService
	interval    time.Duration
	enabled     bool
}

func NewSyncWorker(
	syncRepo *repositories.SyncRepository,
	syncService *services.SyncExecutionService,
	intervalSeconds int,
	enabled bool,
) *SyncWorker {
	return &SyncWorker{
		syncRepo:    syncRepo,
		syncService: syncService,
		interval:    time.Duration(intervalSeconds) * time.Second,
		enabled:     enabled,
	}
}

func (w *SyncWorker) Start(ctx context.Context) {
	if !w.enabled {
		log.Println("Sync Worker is disabled.")
		return
	}

	log.Printf("Starting Sync Worker with interval %v", w.interval)
	ticker := time.NewTicker(w.interval)
	defer ticker.Stop()

	// Initial run after a short delay
	go func() {
		time.Sleep(10 * time.Second)
		w.runPendingJobs()
	}()

	for {
		select {
		case <-ctx.Done():
			log.Println("Stopping Sync Worker...")
			return
		case <-ticker.C:
			w.runPendingJobs()
		}
	}
}

func (w *SyncWorker) runPendingJobs() {
	jobs, err := w.syncRepo.ListSyncJobs("", "", "", "")
	if err != nil {
		log.Printf("Sync Worker: Failed to list jobs: %v", err)
		return
	}

	for _, job := range jobs {
		if !job.IsActive || !job.ScheduleEnabled {
			continue
		}

		// Don't run if already running (with safety timeout)
		if job.Status == "running" {
			if job.LastRunAt != nil && time.Since(*job.LastRunAt) < 30*time.Minute {
				continue
			}
		}

		// Check if it's due
		if job.NextRunAt != nil && time.Now().Before(*job.NextRunAt) {
			continue
		}

		log.Printf("Sync Worker: Executing job %s (%s)", job.JobName, job.ID)

		// Create a copy for the goroutine
		j := job
		go func(jobRec models.SyncJob) {
			defer func() {
				if r := recover(); r != nil {
					log.Printf("Sync Worker: Recovered from panic in job %s: %v", jobRec.ID, r)
				}
			}()

			_, _ = w.syncService.ExecuteJob(&jobRec)

			// Calculate next run time
			interval := 60 // Default 60 mins
			if jobRec.ScheduleIntervalMinutes != nil && *jobRec.ScheduleIntervalMinutes > 0 {
				interval = *jobRec.ScheduleIntervalMinutes
			}

			nextRun := time.Now().Add(time.Duration(interval) * time.Minute)
			jobRec.NextRunAt = &nextRun
			jobRec.Status = "idle"
			_ = w.syncRepo.UpdateSyncJob(&jobRec)

		}(j)
	}
}
