-- Migration: Remove page_size preference from users table
ALTER TABLE users DROP COLUMN IF EXISTS page_size; 