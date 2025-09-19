PRAGMA foreign_keys=OFF;

-- Add o_counter column to galleries table
ALTER TABLE `galleries` ADD COLUMN `o_counter` tinyint not null default 0;

-- Create galleries_o_dates table
CREATE TABLE `galleries_o_dates` (
  `gallery_id` integer not null,
  `o_date` datetime not null,
  foreign key(`gallery_id`) references `galleries`(`id`) on delete CASCADE
);

-- Create index for galleries_o_dates
CREATE INDEX `index_galleries_o_dates` ON `galleries_o_dates` (`gallery_id`);

PRAGMA foreign_keys=ON;

