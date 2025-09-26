ALTER TABLE `performers` ADD COLUMN `primary_tag_id` integer REFERENCES `tags`(`id`) ON DELETE SET NULL;
