-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `users`	ADD COLUMN `num_of_users` INT(11) UNSIGNED NOT NULL DEFAULT '0' AFTER `time_zone`;
ALTER TABLE `users`	ADD COLUMN `admin_email` VARCHAR(255) NOT NULL AFTER `num_of_users`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `users`	DROP COLUMN `num_of_users`;
ALTER TABLE `users`	DROP COLUMN `admin_email`;
