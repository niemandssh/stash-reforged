PRAGMA foreign_keys=OFF;

ALTER TABLE `scenes` ADD COLUMN `embed_url` text;

PRAGMA foreign_keys=ON;
