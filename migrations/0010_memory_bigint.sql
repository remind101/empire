-- +migrate Up
ALTER TABLE processes ALTER COLUMN memory TYPE bigint;

-- +migrate Down
ALTER TABLE processes ALTER COLUMN memory TYPE integer;
