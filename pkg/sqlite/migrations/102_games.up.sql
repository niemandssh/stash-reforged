PRAGMA foreign_keys=OFF;

CREATE TABLE `games` (
  `id` integer not null primary key autoincrement,
  `title` varchar(255) not null,
  `details` text,
  `organized` boolean not null default 0,
  `o_counter` integer not null default 0,
  `omg_counter` integer not null default 0,
  `created_at` datetime not null,
  `updated_at` datetime not null
);

CREATE TABLE `games_o_dates` (
  `game_id` integer not null,
  `o_date` datetime not null,
  foreign key(`game_id`) references `games`(`id`) on delete CASCADE
);

CREATE INDEX `index_games_o_dates` ON `games_o_dates` (`game_id`);

CREATE TABLE `games_omg_dates` (
  `game_id` integer not null,
  `omg_date` datetime not null,
  foreign key(`game_id`) references `games`(`id`) on delete CASCADE
);

CREATE INDEX `index_games_omg_dates` ON `games_omg_dates` (`game_id`);

CREATE TABLE `games_view_dates` (
  `game_id` integer not null,
  `view_date` datetime not null,
  foreign key(`game_id`) references `games`(`id`) on delete CASCADE
);

CREATE INDEX `index_games_view_dates` ON `games_view_dates` (`game_id`);

PRAGMA foreign_keys=ON;


