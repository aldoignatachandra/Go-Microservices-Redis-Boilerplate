-- Initialize single shared database for microservices
-- This script is run when the PostgreSQL container starts

-- Create single shared database
CREATE DATABASE microservices_db;

-- Enable UUID extension
\c microservices_db;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";
