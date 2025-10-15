-- add performer_id column to scenes_tags table
PRAGMA foreign_keys=OFF;

CREATE TABLE `scenes_tags_new` (
  `scene_id` integer,
  `tag_id` integer,
  `performer_id` integer,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`) on delete CASCADE,
  foreign key(`performer_id`) references `performers`(`id`) on delete CASCADE,
  PRIMARY KEY(`scene_id`, `tag_id`, `performer_id`)
);

-- Migrate existing data, setting performer_id to NULL for existing tags
INSERT INTO `scenes_tags_new`
  (
    `scene_id`,
    `tag_id`,
    `performer_id`
  )
  SELECT
    `scene_id`,
    `tag_id`,
    NULL as `performer_id`
  FROM `scenes_tags` WHERE
  `scene_id` IS NOT NULL AND `tag_id` IS NOT NULL;

DROP TABLE `scenes_tags`;
ALTER TABLE `scenes_tags_new` rename to `scenes_tags`;

-- Create indexes
CREATE INDEX `index_scenes_tags_on_tag_id` on `scenes_tags` (`tag_id`);
CREATE INDEX `index_scenes_tags_on_scene_id` on `scenes_tags` (`scene_id`);
CREATE INDEX `index_scenes_tags_on_performer_id` on `scenes_tags` (`performer_id`);

PRAGMA foreign_keys=ON;
