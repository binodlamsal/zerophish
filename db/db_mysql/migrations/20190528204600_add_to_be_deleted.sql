-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `users`	ADD COLUMN `to_be_deleted` TINYINT(1) UNSIGNED NOT NULL DEFAULT '0' AFTER `last_user_agent`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `users`	DROP COLUMN `to_be_deleted`;
