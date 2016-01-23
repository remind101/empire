-- +migrate Up
ALTER TABLE apps DROP COLUMN docker_repo;
ALTER TABLE apps DROP COLUMN github_repo;
ALTER TABLE apps ADD COLUMN repo text;
DROP TABLE deployments;

-- +migrate Down
ALTER TABLE apps DROP COLUMN repo;
ALTER TABLE apps ADD COLUMN docker_repo text;
ALTER TABLE apps ADD COLUMN github_repo text;

CREATE TABLE deployments (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  release_id uuid references releases(id),
  image text NOT NULL,
  status text NOT NULL,
  error text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  finished_at timestamp without time zone
);
