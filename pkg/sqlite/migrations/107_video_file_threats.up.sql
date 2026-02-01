PRAGMA foreign_keys=OFF;

ALTER TABLE `video_files` ADD COLUMN `threats` TEXT;

PRAGMA foreign_keys=ON;
