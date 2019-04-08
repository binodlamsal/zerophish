-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `templates` CHANGE COLUMN `text` `text` BLOB NULL AFTER `subject`;
ALTER TABLE `templates` CHANGE COLUMN `html` `html` MEDIUMBLOB NULL AFTER `text`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `templates` CHANGE COLUMN `text` `text` TEXT NULL AFTER `subject`;
ALTER TABLE `templates` CHANGE COLUMN `html` `html` MEDIUMTEXT NULL AFTER `text`;
