PRAGMA foreign_keys=OFF;

CREATE TABLE `scenes_omg_dates` (
  `scene_id` integer NOT NULL,
  `omg_date` datetime not null,
  foreign key(`scene_id`) references `scenes`(`id`) on delete CASCADE
);

CREATE INDEX `index_scenes_omg_dates` ON `scenes_omg_dates` (`scene_id`);

CREATE TABLE `images_omg_dates` (
  `image_id` integer NOT NULL,
  `omg_date` datetime not null,
  foreign key(`image_id`) references `images`(`id`) on delete CASCADE
);

CREATE INDEX `index_images_omg_dates` ON `images_omg_dates` (`image_id`);

CREATE TABLE `galleries_omg_dates` (
  `gallery_id` integer NOT NULL,
  `omg_date` datetime not null,
  foreign key(`gallery_id`) references `galleries`(`id`) on delete CASCADE
);

CREATE INDEX `index_galleries_omg_dates` ON `galleries_omg_dates` (`gallery_id`);

PRAGMA foreign_keys=ON;

