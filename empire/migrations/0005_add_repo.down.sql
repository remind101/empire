ALTER TABLE apps DROP COLUMN repo;
ALTER TABLE apps ADD COLUMN docker_repo text;
ALTER TABLE apps ADD COLUMN github_repo text;
