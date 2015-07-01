CREATE TABLE certificates (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  name text,
  certificate_chain text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  updated_at timestamp without time zone default (now() at time zone 'utc')
);

CREATE UNIQUE INDEX index_certificates_on_app_id ON certificates USING btree (app_id);
