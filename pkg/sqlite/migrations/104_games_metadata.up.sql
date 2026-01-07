PRAGMA foreign_keys=OFF;

ALTER TABLE `games` ADD COLUMN `date` date;
ALTER TABLE `games` ADD COLUMN `rating` tinyint;

CREATE TABLE `game_urls` (
  `game_id` integer NOT NULL,
  `position` integer NOT NULL,
  `url` varchar(255) NOT NULL,
  foreign key(`game_id`) references `games`(`id`) on delete CASCADE,
  PRIMARY KEY(`game_id`, `position`, `url`)
);

CREATE INDEX `game_urls_url` on `game_urls` (`url`);

CREATE TABLE `games_tags` (
  `game_id` integer NOT NULL,
  `tag_id` integer NOT NULL,
  foreign key(`game_id`) references `games`(`id`) on delete CASCADE,
  foreign key(`tag_id`) references `tags`(`id`) on delete CASCADE,
  PRIMARY KEY(`game_id`, `tag_id`)
);

CREATE INDEX `index_games_tags_on_tag_id` on `games_tags` (`tag_id`);

PRAGMA foreign_keys=ON;



