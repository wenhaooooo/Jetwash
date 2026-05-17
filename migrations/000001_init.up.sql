-- 1. Install pgvector extension (for vector similarity calculation)
CREATE EXTENSION IF NOT EXISTS vector;

-- 2. Create UUID extension (if not already installed)
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

-- Tenants table
CREATE TABLE IF NOT EXISTS tenants (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    parent_id UUID REFERENCES tenants(id) ON DELETE CASCADE,
    api_key VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL UNIQUE,
    email VARCHAR(255) NOT NULL UNIQUE,
    password VARCHAR(255) NOT NULL,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Tenants indexes
CREATE INDEX IF NOT EXISTS idx_tenants_parent_id ON tenants(parent_id);
CREATE INDEX IF NOT EXISTS idx_tenants_status ON tenants(status);
CREATE INDEX IF NOT EXISTS idx_tenants_deleted_at ON tenants(deleted_at);

-- API keys table
CREATE TABLE IF NOT EXISTS api_keys (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    api_key VARCHAR(64) NOT NULL UNIQUE,
    name VARCHAR(255) NOT NULL,
    status SMALLINT DEFAULT 1,
    expires_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- API keys indexes
CREATE INDEX IF NOT EXISTS idx_api_keys_tenant_id ON api_keys(tenant_id);
CREATE INDEX IF NOT EXISTS idx_api_keys_status ON api_keys(status);
CREATE INDEX IF NOT EXISTS idx_api_keys_expires_at ON api_keys(expires_at);
CREATE INDEX IF NOT EXISTS idx_api_keys_deleted_at ON api_keys(deleted_at);

-- Sensitive words table
CREATE TABLE IF NOT EXISTS sensitive_words (
    id SERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    word_text TEXT NOT NULL,
    category VARCHAR(50) NOT NULL,
    risk_level INT NOT NULL DEFAULT 1,
    embedding vector(768) NOT NULL,
    status SMALLINT DEFAULT 1,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Sensitive words indexes
CREATE INDEX IF NOT EXISTS idx_words_tenant_id ON sensitive_words(tenant_id);
CREATE INDEX IF NOT EXISTS idx_words_category ON sensitive_words(category);
CREATE INDEX IF NOT EXISTS idx_words_status ON sensitive_words(status);
CREATE INDEX IF NOT EXISTS idx_words_deleted_at ON sensitive_words(deleted_at);

-- Import tasks table
CREATE TABLE IF NOT EXISTS import_tasks (
    id UUID PRIMARY KEY DEFAULT gen_random_uuid(),
    tenant_id UUID NOT NULL REFERENCES tenants(id) ON DELETE CASCADE,
    file_name VARCHAR(255) NOT NULL,
    status VARCHAR(20) NOT NULL DEFAULT 'pending',
    total INT DEFAULT 0,
    imported INT DEFAULT 0,
    failed INT DEFAULT 0,
    error_msg TEXT,
    started_at TIMESTAMP WITH TIME ZONE,
    completed_at TIMESTAMP WITH TIME ZONE,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Import tasks indexes
CREATE INDEX IF NOT EXISTS idx_import_tenant_id ON import_tasks(tenant_id);
CREATE INDEX IF NOT EXISTS idx_import_status ON import_tasks(status);
CREATE INDEX IF NOT EXISTS idx_import_deleted_at ON import_tasks(deleted_at);

-- Detection history table
CREATE TABLE IF NOT EXISTS detection_history (
    id BIGSERIAL PRIMARY KEY,
    tenant_id UUID NOT NULL,
    text TEXT NOT NULL,
    mode VARCHAR(20) NOT NULL,
    is_offensive BOOLEAN NOT NULL,
    result_json JSON,
    duration BIGINT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    updated_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP,
    deleted_at TIMESTAMP WITH TIME ZONE
);

-- Detection history indexes
CREATE INDEX IF NOT EXISTS idx_history_tenant_id ON detection_history(tenant_id);
CREATE INDEX IF NOT EXISTS idx_history_mode ON detection_history(mode);
CREATE INDEX IF NOT EXISTS idx_history_created_at ON detection_history(created_at);
CREATE INDEX IF NOT EXISTS idx_history_deleted_at ON detection_history(deleted_at);

-- Detection match table
CREATE TABLE IF NOT EXISTS detection_match (
    id BIGSERIAL PRIMARY KEY,
    history_id BIGINT NOT NULL REFERENCES detection_history(id) ON DELETE CASCADE,
    type VARCHAR(50) NOT NULL,
    text VARCHAR(255) NOT NULL,
    confidence FLOAT NOT NULL,
    created_at TIMESTAMP WITH TIME ZONE DEFAULT CURRENT_TIMESTAMP
);

-- Detection match indexes
CREATE INDEX IF NOT EXISTS idx_match_history_id ON detection_match(history_id);

-- Insert default admin tenant (password: admin123)
INSERT INTO tenants (id, api_key, name, email, password, status)
VALUES (
    '00000000-0000-0000-0000-000000000001',
    'jetwash-admin-key-2024',
    'Admin',
    'admin@jetwash.local',
    '$2a$10$N9qo8uLOickgx2ZMRZoMye.IjzqAKL9xL5jvMFVdNJHvGCgTq/VEq',
    1
)
ON CONFLICT (id) DO NOTHING;
