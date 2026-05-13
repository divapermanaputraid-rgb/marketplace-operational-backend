-- Up
CREATE TABLE sync_jobs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID REFERENCES stores(id),
    marketplace VARCHAR(50) NOT NULL, -- shopee, tokopedia_shop, tiktok_shop, all
    sync_type VARCHAR(50) NOT NULL, -- orders, products, inventory, stock, all
    sync_direction VARCHAR(50) NOT NULL, -- pull, push, bidirectional, internal
    job_name VARCHAR(255) NOT NULL,
    status VARCHAR(50) NOT NULL DEFAULT 'idle', -- idle, running, success, failed, skipped, not_configured, disabled
    is_active BOOLEAN NOT NULL DEFAULT true,
    schedule_enabled BOOLEAN NOT NULL DEFAULT false,
    schedule_interval_minutes INTEGER,
    last_run_at TIMESTAMPTZ,
    next_run_at TIMESTAMPTZ,
    last_success_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_sync_jobs_store_id ON sync_jobs(store_id);
CREATE INDEX idx_sync_jobs_marketplace ON sync_jobs(marketplace);
CREATE INDEX idx_sync_jobs_sync_type ON sync_jobs(sync_type);
CREATE INDEX idx_sync_jobs_status ON sync_jobs(status);
CREATE INDEX idx_sync_jobs_deleted_at ON sync_jobs(deleted_at);

CREATE TABLE sync_logs (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    sync_job_id UUID REFERENCES sync_jobs(id) ON DELETE SET NULL,
    store_id UUID REFERENCES stores(id),
    marketplace VARCHAR(50) NOT NULL, -- shopee, tokopedia_shop, tiktok_shop, all
    sync_type VARCHAR(50) NOT NULL, -- orders, products, inventory, stock, all
    sync_direction VARCHAR(50) NOT NULL, -- pull, push, bidirectional, internal
    status VARCHAR(50) NOT NULL, -- started, success, failed, skipped, not_configured
    message TEXT,
    records_processed INTEGER NOT NULL DEFAULT 0,
    records_created INTEGER NOT NULL DEFAULT 0,
    records_updated INTEGER NOT NULL DEFAULT 0,
    records_failed INTEGER NOT NULL DEFAULT 0,
    error_message TEXT,
    started_at TIMESTAMPTZ,
    finished_at TIMESTAMPTZ,
    duration_ms BIGINT,
    raw_summary JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_sync_logs_sync_job_id ON sync_logs(sync_job_id);
CREATE INDEX idx_sync_logs_store_id ON sync_logs(store_id);
CREATE INDEX idx_sync_logs_marketplace ON sync_logs(marketplace);
CREATE INDEX idx_sync_logs_sync_type ON sync_logs(sync_type);
CREATE INDEX idx_sync_logs_status ON sync_logs(status);
CREATE INDEX idx_sync_logs_created_at ON sync_logs(created_at);

-- Down
DROP TABLE IF EXISTS sync_logs;
DROP TABLE IF EXISTS sync_jobs;
