CREATE EXTENSION IF NOT EXISTS hstore;
CREATE EXTENSION IF NOT EXISTS "uuid-ossp";

CREATE TABLE apps (
  name varchar(30) NOT NULL primary key,
  repo text NOT NULL
);

CREATE TABLE configs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name),
  vars hstore
);

CREATE TABLE releases (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name),
  version int NOT NULL
);

CREATE TABLE processes (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  release_id text NOT NULL,
  "type" text NOT NULL,
  quantity int NOT NULL,
  command text NOT NULL
);

CREATE TABLE slugs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  image_repo text NOT NULL,
  image_id text NOT NULL,
  process_types hstore NOT NULL
);

CREATE UNIQUE INDEX index_apps_on_name ON apps USING btree (name);
CREATE UNIQUE INDEX index_processes_on_release_id_and_type ON processes USING btree (release_id, "type");
CREATE UNIQUE INDEX index_slugs_on_image_repo_and_image_id ON slugs USING btree (image_repo, image_id);
CREATE UNIQUE INDEX index_releases_on_app_id_and_version ON releases USING btree (app_id, version);
