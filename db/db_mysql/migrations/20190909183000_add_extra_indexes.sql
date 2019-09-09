-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `group_targets`	ADD INDEX `tid_gid` (`target_id`, `group_id`);
ALTER TABLE `targets`	ADD INDEX `email` (`email`);


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `group_targets`	DROP INDEX `tid_gid`;
ALTER TABLE `targets`	DROP INDEX `email`;
