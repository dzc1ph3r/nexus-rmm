-- Nexus RMM Database Initialization Script
-- This script runs automatically when PostgreSQL container starts for the first time

-- Create extensions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";
CREATE EXTENSION IF NOT EXISTS "pg_trgm";
CREATE EXTENSION IF NOT EXISTS "btree_gin";
CREATE EXTENSION IF NOT EXISTS "btree_gist";

-- Create custom types
DO $$ BEGIN
    CREATE TYPE device_status AS ENUM ('online', 'offline', 'busy', 'maintenance', 'decommissioned');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE user_role AS ENUM ('super_admin', 'org_admin', 'technician', 'viewer', 'api_user');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE task_status AS ENUM ('pending', 'running', 'completed', 'failed', 'cancelled', 'scheduled');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE script_type AS ENUM ('powershell', 'bash', 'python', 'batch', 'cmd');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE alert_severity AS ENUM ('info', 'warning', 'critical', 'emergency');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE session_type AS ENUM ('desktop', 'shell', 'file_transfer', 'command');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

DO $$ BEGIN
    CREATE TYPE session_protocol AS ENUM ('webrtc', 'rdp', 'ssh', 'vnc');
EXCEPTION WHEN duplicate_object THEN NULL; END $$;

-- Create schema for audit logging
CREATE SCHEMA IF NOT EXISTS audit;

-- Function to update updated_at timestamp
CREATE OR REPLACE FUNCTION update_updated_at_column()
RETURNS TRIGGER AS $$
BEGIN
    NEW.updated_at = NOW();
    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Function to generate audit log entry
CREATE OR REPLACE FUNCTION audit.log_change()
RETURNS TRIGGER AS $$
DECLARE
    old_data jsonb;
    new_data jsonb;
BEGIN
    IF TG_OP = 'INSERT' THEN
        old_data = NULL;
        new_data = to_jsonb(NEW);
    ELSIF TG_OP = 'UPDATE' THEN
        old_data = to_jsonb(OLD);
        new_data = to_jsonb(NEW);
    ELSIF TG_OP = 'DELETE' THEN
        old_data = to_jsonb(OLD);
        new_data = NULL;
    END IF;

    INSERT INTO audit.log (table_name, operation, old_data, new_data, changed_by, changed_at)
    VALUES (TG_TABLE_NAME, TG_OP, old_data, new_data, current_setting('app.current_user_id', true)::uuid, NOW());

    RETURN NEW;
END;
$$ LANGUAGE plpgsql;

-- Grant permissions
GRANT USAGE ON SCHEMA audit TO nexus_rmm;
GRANT ALL ON ALL TABLES IN SCHEMA audit TO nexus_rmm;
GRANT ALL ON ALL SEQUENCES IN SCHEMA audit TO nexus_rmm;

-- Set default privileges
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO nexus_rmm;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO nexus_rmm;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT ALL ON TABLES TO nexus_rmm;
ALTER DEFAULT PRIVILEGES IN SCHEMA audit GRANT ALL ON SEQUENCES TO nexus_rmm;

-- Create indexes for better performance (these will be created by Bun migrations, but we prepare here)
-- The actual tables are created by the Go application using Bun ORM migrations

-- Print success message
DO $$
BEGIN
    RAISE NOTICE 'Nexus RMM database initialized successfully!';
END $$;