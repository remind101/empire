CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE deployments (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  status text NOT NULL,
  github_id int NOT NULL,
  sha text NOT NULL,
  ref text NOT NULL,
  environment text NOT NULL,
  description text NOT NULL,
  repo text NOT NULL,
  provider text NOT NULL,
  error text NOT NULL,
  created_at timestamp without time zone default (now() at time zone 'utc') NOT NULL,
  started_at timestamp without time zone,
  completed_at timestamp without time zone
);

CREATE TABLE logs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  deployment_id uuid NOT NULL references deployments(id) ON DELETE CASCADE,
  text text NOT NULL,
  at timestamp without time zone NOT NULL
);

CREATE INDEX index_deployments_on_github_id ON deployments USING btree (github_id);
CREATE INDEX index_deployments_on_created_at ON deployments USING btree (created_at);
CREATE INDEX index_logs_on_at ON logs USING btree (at);
