package models

import (
	"time"

	"github.com/google/uuid"
	"gorm.io/gorm"
)

type SyncJob struct {
	ID                      uuid.UUID      `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	StoreID                 *uuid.UUID     `gorm:"type:uuid;index" json:"store_id,omitempty"`
	Marketplace             string         `gorm:"type:varchar(50);not null;index" json:"marketplace"` // shopee, tokopedia_shop, tiktok_shop, all
	SyncType                string         `gorm:"type:varchar(50);not null;index" json:"sync_type"`   // orders, products, inventory, stock, all
	SyncDirection           string         `gorm:"type:varchar(50);not null" json:"sync_direction"`    // pull, push, bidirectional, internal
	JobName                 string         `gorm:"type:varchar(255);not null" json:"job_name"`
	Status                  string         `gorm:"type:varchar(50);not null;default:'idle';index" json:"status"` // idle, running, success, failed, skipped, not_configured, disabled
	IsActive                bool           `gorm:"type:boolean;not null;default:true" json:"is_active"`
	ScheduleEnabled         bool           `gorm:"type:boolean;not null;default:false" json:"schedule_enabled"`
	ScheduleIntervalMinutes *int           `gorm:"type:integer" json:"schedule_interval_minutes,omitempty"`
	LastRunAt               *time.Time     `gorm:"type:timestamptz" json:"last_run_at,omitempty"`
	NextRunAt               *time.Time     `gorm:"type:timestamptz" json:"next_run_at,omitempty"`
	LastSuccessAt           *time.Time     `gorm:"type:timestamptz" json:"last_success_at,omitempty"`
	LastError               *string        `gorm:"type:text" json:"last_error,omitempty"`
	Config                  *string        `gorm:"type:jsonb" json:"config,omitempty"`
	CreatedAt               time.Time      `gorm:"type:timestamptz;not null;autoCreateTime" json:"created_at"`
	UpdatedAt               time.Time      `gorm:"type:timestamptz;not null;autoUpdateTime" json:"updated_at"`
	DeletedAt               gorm.DeletedAt `gorm:"type:timestamptz;index" json:"-"`

	// Relations
	Store *Store `gorm:"foreignKey:StoreID" json:"store,omitempty"`
}

func (SyncJob) TableName() string {
	return "sync_jobs"
}

type SyncLog struct {
	ID               uuid.UUID  `gorm:"type:uuid;primaryKey;default:gen_random_uuid()" json:"id"`
	SyncJobID        *uuid.UUID `gorm:"type:uuid;index" json:"sync_job_id,omitempty"`
	StoreID          *uuid.UUID `gorm:"type:uuid;index" json:"store_id,omitempty"`
	Marketplace      string     `gorm:"type:varchar(50);not null;index" json:"marketplace"` // shopee, tokopedia_shop, tiktok_shop, all
	SyncType         string     `gorm:"type:varchar(50);not null;index" json:"sync_type"`   // orders, products, inventory, stock, all
	SyncDirection    string     `gorm:"type:varchar(50);not null" json:"sync_direction"`    // pull, push, bidirectional, internal
	Status           string     `gorm:"type:varchar(50);not null;index" json:"status"`      // started, success, failed, skipped, not_configured
	Message          *string    `gorm:"type:text" json:"message,omitempty"`
	RecordsProcessed int        `gorm:"type:integer;not null;default:0" json:"records_processed"`
	RecordsCreated   int        `gorm:"type:integer;not null;default:0" json:"records_created"`
	RecordsUpdated   int        `gorm:"type:integer;not null;default:0" json:"records_updated"`
	RecordsFailed    int        `gorm:"type:integer;not null;default:0" json:"records_failed"`
	ErrorMessage     *string    `gorm:"type:text" json:"error_message,omitempty"`
	StartedAt        *time.Time `gorm:"type:timestamptz" json:"started_at,omitempty"`
	FinishedAt       *time.Time `gorm:"type:timestamptz" json:"finished_at,omitempty"`
	DurationMs       *int64     `gorm:"type:bigint" json:"duration_ms,omitempty"`
	RawSummary       *string    `gorm:"type:jsonb" json:"raw_summary,omitempty"`
	CreatedAt        time.Time  `gorm:"type:timestamptz;not null;autoCreateTime;index" json:"created_at"`

	// Relations
	Store   *Store   `gorm:"foreignKey:StoreID" json:"store,omitempty"`
	SyncJob *SyncJob `gorm:"foreignKey:SyncJobID" json:"sync_job,omitempty"`
}

func (SyncLog) TableName() string {
	return "sync_logs"
}
