package workers

import (
	"context"
	"fmt"
	"log"
	"sync"
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
	runningJobs map[string]bool
	jobsMu      sync.Mutex
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
		runningJobs: make(map[string]bool),
	}
}

func (w *SyncWorker) Start(ctx context.Context) {
	if !w.enabled {
		log.Println("Sync Worker is disabled by configuration.")
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

		// Concurrency guard: don't run if already running in this instance
		w.jobsMu.Lock()
		if w.runningJobs[job.ID.String()] {
			w.jobsMu.Unlock()
			continue
		}

		// Distributed safety: don't run if database says running (with safety timeout)
		if job.Status == "running" {
			if job.LastRunAt != nil && time.Since(*job.LastRunAt) < 30*time.Minute {
				w.jobsMu.Unlock()
				continue
			}
		}

		// Check if it's due
		if job.NextRunAt != nil && time.Now().Before(*job.NextRunAt) {
			w.jobsMu.Unlock()
			continue
		}

		// Mark as running
		w.runningJobs[job.ID.String()] = true
		w.jobsMu.Unlock()

		log.Printf("Sync Worker: Executing job %s (%s)", job.JobName, job.ID)

		// Create a copy for the goroutine
		j := job
		go func(jobRec models.SyncJob) {
			jobID := jobRec.ID.String()
			defer func() {
				w.jobsMu.Lock()
				delete(w.runningJobs, jobID)
				w.jobsMu.Unlock()

				if r := recover(); r != nil {
					log.Printf("Sync Worker: CRITICAL - Recovered from panic in job %s: %v", jobID, r)

					// Reset status so it doesn't get stuck
					jobRec.Status = "failed"
					now := time.Now()
					jobRec.LastRunAt = &now
					errStr := fmt.Sprintf("Recovered from panic: %v", r)
					jobRec.LastError = &errStr
					_ = w.syncRepo.UpdateSyncJob(&jobRec)
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

			// We keep the status from ExecuteJob (success/failed/partial/not_configured/expired)
			// but the worker loop handles moving it to 'idle' or similar if needed for the next run.
			// Actually ExecuteJob already updates the status in DB.
			_ = w.syncRepo.UpdateSyncJob(&jobRec)

		}(j)
	}
}
