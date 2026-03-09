-- Initialize databases for microservices
-- This script creates 3 separate databases (one per service)

-- Create databases (one per service)
CREATE DATABASE auth_db;
CREATE DATABASE user_db;
CREATE DATABASE product_db;

-- Enable UUID extension in each database
\c auth_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c user_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

\c product_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
