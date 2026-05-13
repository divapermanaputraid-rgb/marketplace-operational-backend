-- Migration: 001_create_admins_table
-- Description: Creates the admins table for admin authentication.
-- MVP supports single Admin role with 1-2 admin accounts.

-- Enable UUID extension if not already enabled
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

CREATE TABLE IF NOT EXISTS admins (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    name VARCHAR(120) NOT NULL,
    email VARCHAR(160) UNIQUE NOT NULL,
    password_hash TEXT NOT NULL,
    status VARCHAR(30) NOT NULL DEFAULT 'active',
    last_login_at TIMESTAMPTZ NULL,
    created_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    updated_at TIMESTAMPTZ NOT NULL DEFAULT NOW(),
    deleted_at TIMESTAMPTZ NULL
);

-- Index for soft delete queries
CREATE INDEX IF NOT EXISTS idx_admins_deleted_at ON admins(deleted_at);

-- Index for email lookup (unique index already covers this, but explicit for clarity)
-- The UNIQUE constraint on email already creates an index.

COMMENT ON TABLE admins IS 'Admin user accounts. MVP: single Admin role, 1-2 accounts.';
COMMENT ON COLUMN admins.status IS 'Allowed values: active, inactive';
COMMENT ON COLUMN admins.password_hash IS 'bcrypt hashed password. Never store or return plain text.';
