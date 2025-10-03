-- Add required_for_requirements column to color_presets table
ALTER TABLE color_presets ADD COLUMN required_for_requirements BOOLEAN NOT NULL DEFAULT 1;
