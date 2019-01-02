
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied

CREATE TABLE IF NOT EXISTS `plans` (
	`id` INTEGER PRIMARY KEY AUTO_INCREMENT,
	`name` VARCHAR(255) NOT NULL
);

CREATE TABLE IF NOT EXISTS `subscriptions` (
	`id` INTEGER PRIMARY KEY AUTO_INCREMENT,
	`user_id` INT(10) UNSIGNED NOT NULL,
	`plan_id` INT(10) UNSIGNED NOT NULL,
  `expiration_date` DATETIME NOT NULL
);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back

DROP TABLE plans;
DROP TABLE subscriptions;
