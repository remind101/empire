-- +migrate Up
DROP INDEX index_slugs_on_image;

-- +migrate Down
CREATE UNIQUE INDEX index_slugs_on_image ON slugs USING btree (image);
