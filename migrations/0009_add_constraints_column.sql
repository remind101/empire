-- +migrate Up
ALTER TABLE processes ADD COLUMN cpu_share int;
ALTER TABLE processes ADD COLUMN memory int;

UPDATE processes SET cpu_share = 256, memory = 1073741824;

-- +migrate Down
ALTER TABLE processes DROP COLUMN cpu_share;
ALTER TABLE processes DROP COLUMN memory;
