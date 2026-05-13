-- Up
CREATE TABLE orders (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    store_id UUID NOT NULL REFERENCES stores(id),
    marketplace VARCHAR(50) NOT NULL, -- shopee, tokopedia_shop, tiktok_shop, manual
    external_order_id VARCHAR(255),
    order_number VARCHAR(255) NOT NULL,
    customer_name VARCHAR(255),
    customer_phone VARCHAR(50),
    customer_address TEXT,
    order_status VARCHAR(50) NOT NULL DEFAULT 'pending', -- pending, ready_to_process, packed, shipped, completed, cancelled, returned, failed
    payment_status VARCHAR(50) NOT NULL DEFAULT 'unpaid', -- unpaid, paid, cod, refunded, unknown
    subtotal_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    shipping_fee NUMERIC(12,2) NOT NULL DEFAULT 0,
    discount_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_amount NUMERIC(12,2) NOT NULL DEFAULT 0,
    currency VARCHAR(10) NOT NULL DEFAULT 'IDR',
    notes TEXT,
    ordered_at TIMESTAMPTZ,
    paid_at TIMESTAMPTZ,
    shipped_at TIMESTAMPTZ,
    completed_at TIMESTAMPTZ,
    raw_payload JSONB,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_orders_store_id ON orders(store_id);
CREATE INDEX idx_orders_marketplace ON orders(marketplace);
CREATE INDEX idx_orders_external_order_id ON orders(external_order_id);
CREATE INDEX idx_orders_order_number ON orders(order_number);
CREATE INDEX idx_orders_order_status ON orders(order_status);
CREATE INDEX idx_orders_payment_status ON orders(payment_status);
CREATE INDEX idx_orders_ordered_at ON orders(ordered_at);
CREATE INDEX idx_orders_deleted_at ON orders(deleted_at);

CREATE TABLE order_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    order_id UUID NOT NULL REFERENCES orders(id) ON DELETE CASCADE,
    product_id UUID REFERENCES products(id),
    product_variant_id UUID REFERENCES product_variants(id),
    product_mapping_id UUID REFERENCES marketplace_product_mappings(id),
    sku VARCHAR(255),
    product_name VARCHAR(255) NOT NULL,
    external_product_id VARCHAR(255),
    external_variant_id VARCHAR(255),
    quantity INTEGER NOT NULL CHECK (quantity > 0),
    unit_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    total_price NUMERIC(12,2) NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_order_items_order_id ON order_items(order_id);
CREATE INDEX idx_order_items_product_id ON order_items(product_id);
CREATE INDEX idx_order_items_product_variant_id ON order_items(product_variant_id);
CREATE INDEX idx_order_items_product_mapping_id ON order_items(product_mapping_id);
CREATE INDEX idx_order_items_sku ON order_items(sku);

-- Down
DROP TABLE IF EXISTS order_items;
DROP TABLE IF EXISTS orders;
