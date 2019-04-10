-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `users`	ADD COLUMN `time_zone` VARCHAR(50) NULL AFTER `full_name`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `users`	DROP COLUMN `time_zone`;
