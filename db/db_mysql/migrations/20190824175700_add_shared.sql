-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `templates`	ADD COLUMN `shared` TINYINT(1) NOT NULL DEFAULT 0 AFTER `public`;
ALTER TABLE `pages`	ADD COLUMN `shared` TINYINT(1) NOT NULL DEFAULT 0 AFTER `public`;


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `templates`	DROP COLUMN `shared`;
ALTER TABLE `pages`	DROP COLUMN `shared`;
