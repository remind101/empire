ALTER TABLE processes ADD COLUMN nproc bigint;
UPDATE processes SET nproc = 0;
