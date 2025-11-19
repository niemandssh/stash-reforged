PRAGMA foreign_keys=OFF;

-- Add omg_counter column to scenes table
ALTER TABLE `scenes` ADD COLUMN `omg_counter` tinyint not null default 0;

-- Add omg_counter column to images table
ALTER TABLE `images` ADD COLUMN `omg_counter` tinyint not null default 0;

-- Add omg_counter column to galleries table
ALTER TABLE `galleries` ADD COLUMN `omg_counter` tinyint not null default 0;

PRAGMA foreign_keys=ON;

