-- Add ignore_suggestions field to tags table
ALTER TABLE tags ADD COLUMN ignore_suggestions BOOLEAN NOT NULL DEFAULT 0;
