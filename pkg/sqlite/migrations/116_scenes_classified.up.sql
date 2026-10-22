PRAGMA foreign_keys=OFF;

ALTER TABLE `scenes` ADD COLUMN `classified` boolean not null default '0';

PRAGMA foreign_keys=ON;
