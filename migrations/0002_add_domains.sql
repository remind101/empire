-- +migrate Up
CREATE TABLE domains (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  hostname text NOT NULL,
  created_at timestamp without time zone default (now() at time zone 'utc')
);

CREATE INDEX index_domains_on_app_id ON domains USING btree (app_id);
CREATE UNIQUE INDEX index_domains_on_hostname ON domains USING btree (hostname);

-- +migrate Down
DROP TABLE domains CASCADE;
