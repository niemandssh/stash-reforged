PRAGMA foreign_keys=OFF;

-- Remove omg_counter column from scenes table
-- Note: SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
-- This is a simplified version - in production you'd need to copy all data
CREATE TABLE `scenes_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255),
  `code` text,
  `details` text,
  `director` text,
  `date` date,
  `shoot_date` date,
  `rating` tinyint,
  `organized` boolean not null default '0',
  `pinned` boolean not null default '0',
  `is_broken` boolean not null default '0',
  `is_not_broken` boolean not null default '0',
  `audio_offset_ms` integer not null default 0,
  `audio_playback_speed` real not null default 1.0,
  `force_hls` boolean not null default '0',
  `studio_id` integer,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  `resume_time` real not null default 0,
  `play_duration` real not null default 0,
  `start_time` real,
  `end_time` real,
  `video_filters` text,
  `video_transforms` text,
  `o_counter` tinyint not null default 0,
  foreign key(`studio_id`) references `studios`(`id`) on delete SET NULL
);

-- Copy data without omg_counter column
INSERT INTO `scenes_new`
  SELECT 
    `id`,
    `title`,
    `code`,
    `details`,
    `director`,
    `date`,
    `shoot_date`,
    `rating`,
    `organized`,
    `pinned`,
    `is_broken`,
    `is_not_broken`,
    `audio_offset_ms`,
    `audio_playback_speed`,
    `force_hls`,
    `studio_id`,
    `created_at`,
    `updated_at`,
    `resume_time`,
    `play_duration`,
    `start_time`,
    `end_time`,
    `video_filters`,
    `video_transforms`,
    `o_counter`
  FROM `scenes`;

DROP TABLE `scenes`;
ALTER TABLE `scenes_new` rename to `scenes`;

-- Remove omg_counter column from images table
CREATE TABLE `images_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255),
  `code` text,
  `rating` tinyint,
  `date` date,
  `details` text,
  `photographer` text,
  `organized` boolean not null default '0',
  `o_counter` tinyint not null default 0,
  `studio_id` integer,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  foreign key(`studio_id`) references `studios`(`id`) on delete SET NULL
);

INSERT INTO `images_new`
  SELECT 
    `id`,
    `title`,
    `code`,
    `rating`,
    `date`,
    `details`,
    `photographer`,
    `organized`,
    `o_counter`,
    `studio_id`,
    `created_at`,
    `updated_at`
  FROM `images`;

DROP TABLE `images`;
ALTER TABLE `images_new` rename to `images`;

-- Remove omg_counter column from galleries table
CREATE TABLE `galleries_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255),
  `code` text,
  `date` date,
  `details` text,
  `photographer` text,
  `rating` tinyint,
  `organized` boolean not null default '0',
  `pinned` boolean not null default '0',
  `o_counter` tinyint not null default 0,
  `display_mode` integer not null default 0,
  `created_at` datetime not null,
  `updated_at` datetime not null,
  `studio_id` integer,
  `folder_id` integer,
  foreign key(`studio_id`) references `studios`(`id`) on delete SET NULL,
  foreign key(`folder_id`) references `folders`(`id`) on delete SET NULL
);

INSERT INTO `galleries_new`
  SELECT 
    `id`,
    `title`,
    `code`,
    `date`,
    `details`,
    `photographer`,
    `rating`,
    `organized`,
    `pinned`,
    `o_counter`,
    `display_mode`,
    `created_at`,
    `updated_at`,
    `studio_id`,
    `folder_id`
  FROM `galleries`;

DROP TABLE `galleries`;
ALTER TABLE `galleries_new` rename to `galleries`;

PRAGMA foreign_keys=ON;

