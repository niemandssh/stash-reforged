PRAGMA foreign_keys=OFF;

-- Drop galleries_o_dates table
DROP INDEX IF EXISTS `index_galleries_o_dates`;
DROP TABLE IF EXISTS `galleries_o_dates`;

-- Remove o_counter column from galleries table
-- Note: SQLite doesn't support DROP COLUMN directly, so we need to recreate the table
CREATE TABLE `galleries_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255),
  `code` text,
  `date` date,
  `details` text,
  `photographer` text,
  `rating` tinyint,
  `organized` boolean not null default '0',
  `created_at` datetime not null,
  `updated_at` datetime not null,
  `studio_id` integer,
  `display_mode` varchar(10) not null default 'GRID',
  foreign key(`studio_id`) references `studios`(`id`) on delete SET NULL
);

-- Copy data without o_counter column
INSERT INTO `galleries_new`
  (
    `id`,
    `title`,
    `code`,
    `date`,
    `details`,
    `photographer`,
    `rating`,
    `organized`,
    `created_at`,
    `updated_at`,
    `studio_id`,
    `display_mode`
  )
  SELECT 
    `id`,
    `title`,
    `code`,
    `date`,
    `details`,
    `photographer`,
    `rating`,
    `organized`,
    `created_at`,
    `updated_at`,
    `studio_id`,
    `display_mode`
  FROM `galleries`;

-- Drop old table and rename new one
DROP TABLE `galleries`;
ALTER TABLE `galleries_new` rename to `galleries`;

-- Recreate indexes
CREATE INDEX `index_galleries_on_studio_id` on `galleries` (`studio_id`);

PRAGMA foreign_keys=ON;

