#!/bin/bash

# Fix Migration State Script
# This script fixes "dirty database" issues by resetting and re-running migrations

set -e

echo "🔧 Fixing dirty database migration state..."

# Stop the app container
echo "Stopping app container..."
docker compose stop app

# Connect to database and reset migration state
echo "Resetting migration state in database..."
docker compose exec db psql -U ${DB_USER:-bitcoinpitch} -d ${DB_NAME:-bitcoinpitch} -c "
-- Reset schema_migrations table
DELETE FROM schema_migrations;

-- Optionally, you can also clean the entire database:
-- DROP SCHEMA public CASCADE;
-- CREATE SCHEMA public;
-- GRANT ALL ON SCHEMA public TO ${DB_USER:-bitcoinpitch};
-- GRANT ALL ON SCHEMA public TO public;
"

echo "Migration state reset. Now rebuilding and starting services..."

# Rebuild and restart with fresh migrations
docker compose build --no-cache app
docker compose up -d

echo "✅ Database migration issue fixed!"
echo "Check logs: docker compose logs -f app" 