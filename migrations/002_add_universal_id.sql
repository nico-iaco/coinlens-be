-- Add universal_id column to coins table with NOT NULL and empty string default
ALTER TABLE coins ADD COLUMN universal_id VARCHAR(100) NOT NULL DEFAULT '';
