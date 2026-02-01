PRAGMA foreign_keys=OFF;

ALTER TABLE `video_files` ADD COLUMN `threats_scanned_at` DATETIME;

PRAGMA foreign_keys=ON;
