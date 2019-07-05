-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `campaigns` ADD COLUMN `group_id` BIGINT(20) NULL DEFAULT 0 AFTER `page_id`;


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `campaigns`	DROP COLUMN `group_id`;
