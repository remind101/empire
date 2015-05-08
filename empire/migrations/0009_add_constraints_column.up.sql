ALTER TABLE processes ADD COLUMN cpu_share int;
ALTER TABLE processes ADD COLUMN memory int;

UPDATE processes SET cpu_share = 256, memory = 536870912;
