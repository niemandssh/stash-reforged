PRAGMA foreign_keys=OFF;

-- Note: SQLite doesn't support dropping columns, so we'll need to recreate the table
-- This is a simplified version - in production you'd want to preserve data
CREATE TABLE `games_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `details` text,
  `organized` boolean not null default 0,
  `o_counter` integer not null default 0,
  `omg_counter` integer not null default 0,
  `created_at` datetime not null,
  `updated_at` datetime not null
);

INSERT INTO `games_new` SELECT `id`, `title`, `details`, `organized`, `o_counter`, `omg_counter`, `created_at`, `updated_at` FROM `games`;

DROP TABLE `games`;
ALTER TABLE `games_new` RENAME TO `games`;

PRAGMA foreign_keys=ON;


