-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `templates` ADD COLUMN `from_address` VARCHAR(255) NULL DEFAULT NULL AFTER `html`;
ALTER TABLE `campaigns` ADD COLUMN `from_address` VARCHAR(255) NULL DEFAULT NULL AFTER `time_zone`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
