-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `pages` CHANGE COLUMN `html` `html` LONGBLOB NULL AFTER `name`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `pages` CHANGE COLUMN `html` `html` LONGTEXT NULL AFTER `name`;
