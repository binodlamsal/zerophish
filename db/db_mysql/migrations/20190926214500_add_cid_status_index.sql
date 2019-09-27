-- +goose Up
-- SQL in this section is executed when the migration is applied.

ALTER TABLE `results`	ADD INDEX `cid_status` (`campaign_id`, `status`);


-- +goose Down
-- SQL in this section is executed when the migration is rolled back.

ALTER TABLE `results`	DROP INDEX `cid_status`;
