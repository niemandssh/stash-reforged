-- remove performer_id column from scenes_tags table
PRAGMA foreign_keys=OFF;

CREATE TABLE `scenes_tags_old` (
  `scene_id` integer,
  `tag_id` integer,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`) on delete CASCADE,
  PRIMARY KEY(`scene_id`, `tag_id`)
);

-- Migrate data back, keeping only tags that are not associated with performers
INSERT INTO `scenes_tags_old`
  (
    `scene_id`,
    `tag_id`
  )
  SELECT
    `scene_id`,
    `tag_id`
  FROM `scenes_tags` WHERE
  `scene_id` IS NOT NULL AND `tag_id` IS NOT NULL AND `performer_id` IS NULL;

DROP TABLE `scenes_tags`;
ALTER TABLE `scenes_tags_old` rename to `scenes_tags`;

-- Recreate indexes
CREATE INDEX `index_scenes_tags_on_tag_id` on `scenes_tags` (`tag_id`);
CREATE INDEX `index_scenes_tags_on_scene_id` on `scenes_tags` (`scene_id`);

PRAGMA foreign_keys=ON;
