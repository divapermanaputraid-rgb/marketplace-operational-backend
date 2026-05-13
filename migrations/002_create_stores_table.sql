-- Up
CREATE TABLE stores (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    marketplace VARCHAR(50) NOT NULL,
    store_name VARCHAR(255) NOT NULL,
    store_url TEXT,
    external_store_id VARCHAR(255),
    connection_status VARCHAR(50) NOT NULL DEFAULT 'not_connected',
    is_active BOOLEAN NOT NULL DEFAULT true,
    notes TEXT,
    last_sync_at TIMESTAMPTZ,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_stores_marketplace ON stores(marketplace);
CREATE INDEX idx_stores_external_store_id ON stores(external_store_id);
CREATE INDEX idx_stores_deleted_at ON stores(deleted_at);

CREATE TABLE marketplace_connections (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id) ON DELETE CASCADE,
    marketplace VARCHAR(50) NOT NULL,
    connection_status VARCHAR(50) NOT NULL DEFAULT 'not_connected',
    access_token_encrypted TEXT,
    refresh_token_encrypted TEXT,
    token_expires_at TIMESTAMPTZ,
    scopes TEXT,
    connected_at TIMESTAMPTZ,
    disconnected_at TIMESTAMPTZ,
    last_error TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE UNIQUE INDEX idx_marketplace_connections_store_id ON marketplace_connections(store_id);

-- Down
DROP TABLE IF EXISTS marketplace_connections;
DROP TABLE IF EXISTS stores;
