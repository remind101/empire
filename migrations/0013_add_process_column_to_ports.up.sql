ALTER TABLE ports ADD COLUMN taken text;
UPDATE ports SET taken = 't' WHERE port = (SELECT port FROM ports WHERE app_id != NULL);
