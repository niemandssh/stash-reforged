PRAGMA foreign_keys=OFF;

-- Add is_broken column to scenes table
ALTER TABLE `scenes` ADD COLUMN `is_broken` boolean not null default '0';

PRAGMA foreign_keys=ON;
