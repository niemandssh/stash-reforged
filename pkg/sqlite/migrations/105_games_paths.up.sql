PRAGMA foreign_keys=OFF;

ALTER TABLE `games` ADD COLUMN `folder_path` varchar(255);
ALTER TABLE `games` ADD COLUMN `executable_path` varchar(255);

PRAGMA foreign_keys=ON;


