-- Add video filters and transforms columns to scenes table
ALTER TABLE `scenes` ADD COLUMN `video_filters` TEXT;
ALTER TABLE `scenes` ADD COLUMN `video_transforms` TEXT;
