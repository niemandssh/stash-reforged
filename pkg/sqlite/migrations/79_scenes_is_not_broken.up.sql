PRAGMA foreign_keys=OFF;

-- Add is_not_broken column to scenes table
ALTER TABLE `scenes` ADD COLUMN `is_not_broken` boolean not null default '0';

PRAGMA foreign_keys=ON;
