-- Drop scene_similarities table and related objects
DROP TRIGGER IF EXISTS scene_similarities_updated_at;
DROP INDEX IF EXISTS idx_scene_similarities_score;
DROP INDEX IF EXISTS idx_scene_similarities_similar_scene_id;
DROP INDEX IF EXISTS idx_scene_similarities_scene_id;
DROP TABLE IF EXISTS scene_similarities;
