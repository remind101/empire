-- Values: private, public
ALTER TABLE apps ADD COLUMN exposure TEXT NOT NULL default 'private';