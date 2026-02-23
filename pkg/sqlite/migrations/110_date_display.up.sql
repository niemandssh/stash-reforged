-- Store how to display partial dates: "YYYY" (year only), "YYYY-MM" (year+month), or NULL (full date).
ALTER TABLE `performers` ADD COLUMN `birthdate_display` TEXT;
ALTER TABLE `performers` ADD COLUMN `death_date_display` TEXT;
ALTER TABLE `scenes` ADD COLUMN `date_display` TEXT;
ALTER TABLE `scenes` ADD COLUMN `shoot_date_display` TEXT;
ALTER TABLE `images` ADD COLUMN `date_display` TEXT;
ALTER TABLE `galleries` ADD COLUMN `date_display` TEXT;
ALTER TABLE `games` ADD COLUMN `date_display` TEXT;
ALTER TABLE `groups` ADD COLUMN `date_display` TEXT;
