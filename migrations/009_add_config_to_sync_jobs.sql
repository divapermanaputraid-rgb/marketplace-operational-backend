-- Up
ALTER TABLE sync_jobs ADD COLUMN config JSONB;

-- Down
ALTER TABLE sync_jobs DROP COLUMN config;
