PRAGMA foreign_keys=OFF;

-- Note: SQLite doesn't support dropping columns, so we'll need to recreate the table
CREATE TABLE `games_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `details` text,
  `date` date,
  `rating` tinyint,
  `organized` boolean not null default 0,
  `o_counter` integer not null default 0,
  `omg_counter` integer not null default 0,
  `image` blob,
  `created_at` datetime not null,
  `updated_at` datetime not null
);

INSERT INTO `games_new`
  (
    `id`,
    `title`,
    `details`,
    `date`,
    `rating`,
    `organized`,
    `o_counter`,
    `omg_counter`,
    `image`,
    `created_at`,
    `updated_at`
  )
  SELECT
    `id`,
    `title`,
    `details`,
    `date`,
    `rating`,
    `organized`,
    `o_counter`,
    `omg_counter`,
    `image`,
    `created_at`,
    `updated_at`
  FROM `games`;

DROP TABLE `games`;
ALTER TABLE `games_new` RENAME TO `games`;

PRAGMA foreign_keys=ON;


