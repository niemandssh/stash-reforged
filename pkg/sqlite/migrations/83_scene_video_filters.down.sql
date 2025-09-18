-- Remove video filters and transforms columns from scenes table
ALTER TABLE `scenes` DROP COLUMN `video_filters`;
ALTER TABLE `scenes` DROP COLUMN `video_transforms`;
