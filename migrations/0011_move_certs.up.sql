ALTER TABLE apps ADD COLUMN cert text;
UPDATE apps SET cert = (select name from certificates where certificates.app_id = apps.id);
