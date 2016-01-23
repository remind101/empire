-- +migrate Up
-- Values: private, public
ALTER TABLE apps ADD COLUMN exposure TEXT NOT NULL default 'private';
-- +migrate Down
ALTER TABLE apps DROP COLUMN exposure;
