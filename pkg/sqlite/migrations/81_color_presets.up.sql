-- Create color_presets table
CREATE TABLE color_presets (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    name TEXT NOT NULL,
    color TEXT NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP
);

-- Create index on name for faster lookups
CREATE INDEX idx_color_presets_name ON color_presets(name);

-- Create index on color for faster lookups
CREATE INDEX idx_color_presets_color ON color_presets(color);
