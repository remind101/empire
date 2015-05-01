ALTER TABLE apps DROP COLUMN docker_repo;
ALTER TABLE apps DROP COLUMN github_repo;
ALTER TABLE apps ADD COLUMN repo text;
