PRAGMA foreign_keys=OFF;

ALTER TABLE `scenes` ADD COLUMN `is_ai_filled` boolean not null default '0';

PRAGMA foreign_keys=ON;
