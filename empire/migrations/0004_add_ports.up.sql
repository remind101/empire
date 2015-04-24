CREATE TABLE ports (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  port integer,
  app_id text references apps(name) ON DELETE SET NULL
);

-- Insert IANA suggested private port range.
INSERT INTO ports (port) (SELECT generate_series(49152,65535));