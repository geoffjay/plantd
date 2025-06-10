-- Initialize the Identity Service Database
-- This script is run when the PostgreSQL container starts for the first time

-- Create database if it doesn't exist (already created by POSTGRES_DB env var)
-- Create extensions for UUID and crypto functions
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
CREATE EXTENSION IF NOT EXISTS "pgcrypto";

-- Grant all privileges to the identity user
GRANT ALL PRIVILEGES ON DATABASE plantd_identity TO identity_user;
GRANT ALL PRIVILEGES ON ALL TABLES IN SCHEMA public TO identity_user;
GRANT ALL PRIVILEGES ON ALL SEQUENCES IN SCHEMA public TO identity_user;
GRANT ALL PRIVILEGES ON ALL FUNCTIONS IN SCHEMA public TO identity_user;

-- Set default privileges for future objects
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON TABLES TO identity_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON SEQUENCES TO identity_user;
ALTER DEFAULT PRIVILEGES IN SCHEMA public GRANT ALL ON FUNCTIONS TO identity_user;

-- Log successful initialization
SELECT 'Identity Service database initialized successfully' AS status; 
