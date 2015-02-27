CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE apps (
  name varchar(30) NOT NULL primary key,
  repo text NOT NULL
);

CREATE TABLE configs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  vars hstore,
  created_at timestamp without time zone default (now() at time zone 'utc')
);

CREATE TABLE slugs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  image text NOT NULL,
  process_types hstore NOT NULL
);

CREATE TABLE releases (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  config_id uuid NOT NULL references configs(id) ON DELETE CASCADE,
  slug_id uuid NOT NULL references slugs(id) ON DELETE CASCADE,
  version int NOT NULL,
  created_at timestamp without time zone default (now() at time zone 'utc')
);

CREATE TABLE processes (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  release_id uuid NOT NULL references releases(id) ON DELETE CASCADE,
  "type" text NOT NULL,
  quantity int NOT NULL,
  command text NOT NULL
);

CREATE TABLE jobs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  release_version int NOT NULL,
  process_type text NOT NULL,
  instance int NOT NULL,

  environment hstore NOT NULL,
  image text NOT NULL,
  command text NOT NULL
);

CREATE UNIQUE INDEX index_apps_on_name ON apps USING btree (name);
CREATE UNIQUE INDEX index_processes_on_release_id_and_type ON processes USING btree (release_id, "type");
CREATE UNIQUE INDEX index_slugs_on_image ON slugs USING btree (image);
CREATE UNIQUE INDEX index_releases_on_app_id_and_version ON releases USING btree (app_id, version);
CREATE INDEX index_configs_on_created_at ON configs (created_at);
