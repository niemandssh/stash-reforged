-- Add performer_ids to scenes_o_dates and scenes_omg_dates.
-- performer_ids is a JSON array of performer IDs (e.g. "[1,2,3]") or NULL for "attribute to whole scene".
ALTER TABLE `scenes_o_dates` ADD COLUMN `performer_ids` TEXT;
ALTER TABLE `scenes_omg_dates` ADD COLUMN `performer_ids` TEXT;
