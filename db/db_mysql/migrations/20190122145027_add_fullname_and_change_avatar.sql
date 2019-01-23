
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `users` ADD `full_name` VARCHAR(255) NOT NULL AFTER `api_key`;
ALTER TABLE `users` MODIFY `avatar` BLOB;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `users` DROP `full_name`;
ALTER TABLE `users` MODIFY `avatar` VARCHAR(191) NOT NULL;
