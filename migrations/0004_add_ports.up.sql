CREATE TABLE ports (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  port integer,
  app_id uuid references apps(id) ON DELETE SET NULL
);

-- Insert 1000 ports
INSERT INTO ports (port) (SELECT generate_series(9000,10000));
