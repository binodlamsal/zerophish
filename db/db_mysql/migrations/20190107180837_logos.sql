
-- +goose Up
-- SQL in section 'Up' is executed when this migration is applied
CREATE TABLE IF NOT EXISTS logos (id integer primary key auto_increment,user_id int(10) unsigned not null,data blob not null);

-- +goose Down
-- SQL section 'Down' is executed when this migration is rolled back
DROP TABLE logos;
