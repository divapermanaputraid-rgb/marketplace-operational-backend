-- Sprint 13: Marketplace Database & Backend Prep
-- Migration 008: Create marketplace credentials and OAuth state tables

-- marketplace_credentials: stores encrypted OAuth tokens per store+marketplace pair
CREATE TABLE IF NOT EXISTS marketplace_credentials (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    marketplace VARCHAR(50) NOT NULL,
    connection_status VARCHAR(50) NOT NULL DEFAULT 'not_configured',
    app_id VARCHAR(255),
    encrypted_access_token TEXT,
    encrypted_refresh_token TEXT,
    access_token_expires_at TIMESTAMPTZ,
    refresh_token_expires_at TIMESTAMPTZ,
    scopes TEXT,
    last_connected_at TIMESTAMPTZ,
    last_refreshed_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

-- Unique constraint: one active credential per store + marketplace (soft delete aware)
CREATE UNIQUE INDEX IF NOT EXISTS idx_cred_store_marketplace
    ON marketplace_credentials (store_id, marketplace)
    WHERE deleted_at IS NULL;

CREATE INDEX IF NOT EXISTS idx_cred_marketplace ON marketplace_credentials (marketplace);
CREATE INDEX IF NOT EXISTS idx_cred_connection_status ON marketplace_credentials (connection_status);
CREATE INDEX IF NOT EXISTS idx_cred_deleted_at ON marketplace_credentials (deleted_at);

-- marketplace_oauth_states: tracks OAuth authorization flow state for CSRF prevention
CREATE TABLE IF NOT EXISTS marketplace_oauth_states (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    state VARCHAR(255) NOT NULL,
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    marketplace VARCHAR(50) NOT NULL,
    redirect_uri TEXT,
    status VARCHAR(50) NOT NULL DEFAULT 'pending',
    expires_at TIMESTAMPTZ NOT NULL,
    used_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX IF NOT EXISTS idx_oauth_state ON marketplace_oauth_states (state);
CREATE INDEX IF NOT EXISTS idx_oauth_state_store ON marketplace_oauth_states (store_id);
CREATE INDEX IF NOT EXISTS idx_oauth_state_expires ON marketplace_oauth_states (expires_at);
