-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `campaigns` ADD COLUMN `remove_non_clickers` TINYINT(1) NOT NULL DEFAULT '0' AFTER `from_address`;
ALTER TABLE `campaigns` ADD COLUMN `clickers_group_id` BIGINT(20) NULL DEFAULT NULL AFTER `remove_non_clickers`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `campaigns`	DROP COLUMN `clickers_group_id`;
ALTER TABLE `campaigns`	DROP COLUMN `remove_non_clickers`;
