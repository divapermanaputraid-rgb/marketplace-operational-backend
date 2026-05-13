-- Up
CREATE TABLE marketplace_product_mappings (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    product_variant_id UUID REFERENCES product_variants(id),
    store_id UUID NOT NULL REFERENCES stores(id),
    marketplace VARCHAR(50) NOT NULL,
    external_product_id VARCHAR(255) NOT NULL,
    external_variant_id VARCHAR(255),
    external_sku VARCHAR(255),
    listing_name VARCHAR(255),
    listing_url TEXT,
    listing_status VARCHAR(50) NOT NULL DEFAULT 'unknown',
    price NUMERIC(12,2),
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    last_synced_at TIMESTAMPTZ,
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_marketplace_product_mappings_product_id ON marketplace_product_mappings(product_id);
CREATE INDEX idx_marketplace_product_mappings_product_variant_id ON marketplace_product_mappings(product_variant_id);
CREATE INDEX idx_marketplace_product_mappings_store_id ON marketplace_product_mappings(store_id);
CREATE INDEX idx_marketplace_product_mappings_marketplace ON marketplace_product_mappings(marketplace);
CREATE INDEX idx_marketplace_product_mappings_external_product_id ON marketplace_product_mappings(external_product_id);
CREATE INDEX idx_marketplace_product_mappings_listing_status ON marketplace_product_mappings(listing_status);
CREATE INDEX idx_marketplace_product_mappings_deleted_at ON marketplace_product_mappings(deleted_at);

-- Unique constraint for duplicate prevention (handling NULL external_variant_id)
CREATE UNIQUE INDEX idx_marketplace_product_mappings_unique 
ON marketplace_product_mappings (store_id, external_product_id, COALESCE(external_variant_id, '')) 
WHERE deleted_at IS NULL;

-- Down
DROP INDEX IF EXISTS idx_marketplace_product_mappings_unique;
DROP TABLE IF EXISTS marketplace_product_mappings;
