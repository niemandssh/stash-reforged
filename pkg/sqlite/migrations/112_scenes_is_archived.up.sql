PRAGMA foreign_keys=OFF;

ALTER TABLE `scenes` ADD COLUMN `is_archived` boolean not null default '0';
ALTER TABLE `scenes` ADD COLUMN `archive_reason` text;

PRAGMA foreign_keys=ON;
