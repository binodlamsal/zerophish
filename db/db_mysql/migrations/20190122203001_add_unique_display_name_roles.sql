-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `roles` ADD `display_name` VARCHAR(50) NOT NULL AFTER `name`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
