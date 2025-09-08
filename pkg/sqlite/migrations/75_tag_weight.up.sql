-- Add weight column to tags table
ALTER TABLE tags ADD COLUMN weight REAL NOT NULL DEFAULT 0.5;

-- Update existing tags to have default weight of 0.5
UPDATE tags SET weight = 0.5 WHERE weight IS NULL;
