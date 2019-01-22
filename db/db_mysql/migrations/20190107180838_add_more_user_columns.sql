-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `users` ADD `avatar` VARCHAR(191) NOT NULL AFTER `api_key`, ADD `email_verified_at` TIMESTAMP NULL AFTER `avatar`, ADD `created_at` TIMESTAMP NULL AFTER `email_verified_at`, ADD `updated_at` TIMESTAMP NULL AFTER `created_at`, ADD `last_login_at` TIMESTAMP NULL AFTER `updated_at`, ADD `last_login_ip` VARCHAR(191) NOT NULL AFTER `last_login_at`;

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.
