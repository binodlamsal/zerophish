-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `users` ADD `last_user_agent` VARCHAR(255) NULL DEFAULT NULL AFTER `last_login_ip`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
