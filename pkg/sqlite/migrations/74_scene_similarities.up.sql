-- Create scene_similarities table
CREATE TABLE scene_similarities (
    id INTEGER PRIMARY KEY AUTOINCREMENT,
    scene_id INTEGER NOT NULL,
    similar_scene_id INTEGER NOT NULL,
    similarity_score REAL NOT NULL,
    created_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    updated_at DATETIME NOT NULL DEFAULT CURRENT_TIMESTAMP,
    FOREIGN KEY (scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
    FOREIGN KEY (similar_scene_id) REFERENCES scenes(id) ON DELETE CASCADE,
    UNIQUE(scene_id, similar_scene_id)
);

-- Create indexes for better performance
CREATE INDEX idx_scene_similarities_scene_id ON scene_similarities(scene_id);
CREATE INDEX idx_scene_similarities_similar_scene_id ON scene_similarities(similar_scene_id);
CREATE INDEX idx_scene_similarities_score ON scene_similarities(similar_scene_id, similarity_score DESC);

-- Create trigger to update updated_at timestamp
CREATE TRIGGER scene_similarities_updated_at 
    AFTER UPDATE ON scene_similarities
    FOR EACH ROW
BEGIN
    UPDATE scene_similarities 
    SET updated_at = CURRENT_TIMESTAMP 
    WHERE id = NEW.id;
END;
