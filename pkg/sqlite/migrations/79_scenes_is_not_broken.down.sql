PRAGMA foreign_keys=OFF;

-- Remove is_not_broken column from scenes table
ALTER TABLE `scenes` DROP COLUMN `is_not_broken`;

PRAGMA foreign_keys=ON;
