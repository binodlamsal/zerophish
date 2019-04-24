-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `templates`	ADD COLUMN `default_page_id` INT(10) UNSIGNED NULL AFTER `tag`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `templates`	DROP COLUMN `default_page_id`;
