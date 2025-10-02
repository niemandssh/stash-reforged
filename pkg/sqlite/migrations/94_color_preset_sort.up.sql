-- Add sort column to color_presets table
ALTER TABLE color_presets ADD COLUMN sort INTEGER NOT NULL DEFAULT 0;
