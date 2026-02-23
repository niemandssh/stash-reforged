-- Requires SQLite 3.35.0+ (DROP COLUMN). go-sqlite3 v1.14+ ships with SQLite 3.45+.
ALTER TABLE `scenes_o_dates` DROP COLUMN `performer_ids`;
ALTER TABLE `scenes_omg_dates` DROP COLUMN `performer_ids`;
