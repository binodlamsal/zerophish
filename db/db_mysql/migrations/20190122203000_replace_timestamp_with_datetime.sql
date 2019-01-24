-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
ALTER TABLE `users`
  MODIFY `email_verified_at` DATETIME NULL,
  MODIFY `created_at` DATETIME NULL,
  MODIFY `updated_at` DATETIME NULL,
  MODIFY `last_login_at` DATETIME NULL;

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
ALTER TABLE `users`
  MODIFY `email_verified_at` TIMESTAMP NULL,
  MODIFY `created_at` TIMESTAMP NULL,
  MODIFY `updated_at` TIMESTAMP NULL,
  MODIFY `last_login_at` TIMESTAMP NULL;
