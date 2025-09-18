-- Create table for multiple performer profile images
CREATE TABLE `performer_profile_images` (
  `id` integer PRIMARY KEY AUTOINCREMENT,
  `performer_id` integer NOT NULL,
  `image_blob` varchar(255) REFERENCES `blobs`(`checksum`),
  `is_primary` boolean NOT NULL DEFAULT 0,
  `position` integer NOT NULL DEFAULT 0,
  `created_at` datetime NOT NULL,
  `updated_at` datetime NOT NULL,
  FOREIGN KEY (`performer_id`) REFERENCES `performers` (`id`) ON DELETE CASCADE
);

-- Create indexes for better performance
CREATE INDEX `idx_performer_profile_images_performer_id` ON `performer_profile_images` (`performer_id`);
CREATE INDEX `idx_performer_profile_images_is_primary` ON `performer_profile_images` (`performer_id`, `is_primary`);
CREATE INDEX `idx_performer_profile_images_position` ON `performer_profile_images` (`performer_id`, `position`);

-- Migrate existing performer images to the new table
INSERT INTO `performer_profile_images` (`performer_id`, `image_blob`, `is_primary`, `position`, `created_at`, `updated_at`)
SELECT 
  `id` as `performer_id`,
  `image_blob`,
  1 as `is_primary`,
  0 as `position`,
  COALESCE(`created_at`, datetime('now')) as `created_at`,
  COALESCE(`updated_at`, datetime('now')) as `updated_at`
FROM `performers` 
WHERE `image_blob` IS NOT NULL AND `image_blob` != '';
