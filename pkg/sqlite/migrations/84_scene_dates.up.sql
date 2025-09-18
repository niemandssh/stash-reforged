-- Add shoot_date column for filming/shooting date
-- Keep existing date column as release date
ALTER TABLE `scenes` ADD COLUMN `shoot_date` date;
