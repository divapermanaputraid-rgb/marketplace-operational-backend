-- Up
CREATE TABLE inventory_items (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    product_id UUID NOT NULL REFERENCES products(id),
    product_variant_id UUID REFERENCES product_variants(id),
    sku VARCHAR(255) NOT NULL,
    location_name VARCHAR(255) NOT NULL DEFAULT 'Main Warehouse',
    available_quantity INTEGER NOT NULL DEFAULT 0,
    reserved_quantity INTEGER NOT NULL DEFAULT 0,
    damaged_quantity INTEGER NOT NULL DEFAULT 0,
    safety_stock INTEGER NOT NULL DEFAULT 0,
    notes TEXT,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ
);

CREATE INDEX idx_inventory_items_product_id ON inventory_items(product_id);
CREATE INDEX idx_inventory_items_product_variant_id ON inventory_items(product_variant_id);
CREATE INDEX idx_inventory_items_sku ON inventory_items(sku);
CREATE INDEX idx_inventory_items_deleted_at ON inventory_items(deleted_at);
-- Unique constraint for product + variant + location
CREATE UNIQUE INDEX idx_inventory_items_unique_item ON inventory_items (product_id, COALESCE(product_variant_id, '00000000-0000-0000-0000-000000000000'), location_name) WHERE deleted_at IS NULL;

CREATE TABLE inventory_movements (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    inventory_item_id UUID NOT NULL REFERENCES inventory_items(id),
    product_id UUID NOT NULL REFERENCES products(id),
    product_variant_id UUID REFERENCES product_variants(id),
    movement_type VARCHAR(50) NOT NULL, -- initial, adjustment_in, adjustment_out, reserved, reservation_released, sold, returned, damaged, manual_correction
    quantity_delta INTEGER NOT NULL,
    quantity_before INTEGER NOT NULL,
    quantity_after INTEGER NOT NULL,
    reference_type VARCHAR(100),
    reference_id VARCHAR(100),
    notes TEXT,
    created_by_admin_id UUID REFERENCES admins(id),
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW()
);

CREATE INDEX idx_inventory_movements_inventory_item_id ON inventory_movements(inventory_item_id);
CREATE INDEX idx_inventory_movements_product_id ON inventory_movements(product_id);
CREATE INDEX idx_inventory_movements_product_variant_id ON inventory_movements(product_variant_id);
CREATE INDEX idx_inventory_movements_movement_type ON inventory_movements(movement_type);
CREATE INDEX idx_inventory_movements_created_at ON inventory_movements(created_at);

-- Down
DROP TABLE IF EXISTS inventory_movements;
DROP TABLE IF EXISTS inventory_items;
