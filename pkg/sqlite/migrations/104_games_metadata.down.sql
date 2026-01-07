PRAGMA foreign_keys=OFF;

DROP TABLE IF EXISTS `games_tags`;
DROP INDEX IF EXISTS `game_urls_url`;
DROP TABLE IF EXISTS `game_urls`;

CREATE TABLE `games_new` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `details` text,
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



