PRAGMA foreign_keys=OFF;

-- Create galleries_view_dates table
CREATE TABLE `galleries_view_dates` (
  `gallery_id` integer not null,
  `view_date` datetime not null,
  foreign key(`gallery_id`) references `galleries`(`id`) on delete CASCADE
);

-- Create index for galleries_view_dates
CREATE INDEX `index_galleries_view_dates` ON `galleries_view_dates` (`gallery_id`);

PRAGMA foreign_keys=ON;
