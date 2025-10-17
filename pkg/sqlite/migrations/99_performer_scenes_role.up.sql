-- Add small_role and role_description columns to existing performers_scenes table
ALTER TABLE `performers_scenes` ADD COLUMN `small_role` boolean not null default '0';
ALTER TABLE `performers_scenes` ADD COLUMN `role_description` varchar(255);
