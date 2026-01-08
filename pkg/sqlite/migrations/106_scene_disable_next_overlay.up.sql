PRAGMA foreign_keys=OFF;

ALTER TABLE `scenes` ADD COLUMN `disable_next_scene_overlay` BOOLEAN DEFAULT 0;

PRAGMA foreign_keys=ON;
