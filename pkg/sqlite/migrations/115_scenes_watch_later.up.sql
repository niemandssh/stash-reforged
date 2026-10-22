CREATE TABLE `scenes_watch_later` (
  `scene_id` integer NOT NULL UNIQUE,
  `added_at` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);
CREATE INDEX `scenes_watch_later_added_at` on `scenes_watch_later` (`added_at`);
