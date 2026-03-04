-- Initialize databases for microservices
-- This script is run when the PostgreSQL container starts

-- Create databases for each service
CREATE DATABASE auth_db;
CREATE DATABASE user_db;
CREATE DATABASE product_db;

-- Enable UUID extension in all databases
\c auth_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c user_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c product_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
