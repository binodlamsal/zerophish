-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `logos`	CHANGE COLUMN `data` `data` MEDIUMBLOB NOT NULL AFTER `user_id`;
ALTER TABLE `users`	DROP COLUMN `avatar`;
CREATE TABLE IF NOT EXISTS avatars (id integer primary key auto_increment,user_id int(10) unsigned not null,data mediumblob not null);

-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `logos`	CHANGE COLUMN `data` `data` BLOB NOT NULL AFTER `user_id`;
ALTER TABLE `users`	ADD COLUMN `avatar_id` BLOB NOT NULL AFTER `full_name`;
DROP TABLE avatars;
