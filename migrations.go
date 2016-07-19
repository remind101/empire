package empire

import (
	"database/sql"
	"encoding/json"

	"github.com/lib/pq/hstore"
	"github.com/remind101/empire/pkg/constraints"
	"github.com/remind101/empire/procfile"
	"github.com/remind101/migrate"
)

var migrations = []migrate.Migration{
	{
		ID: 1,
		Up: migrate.Queries([]string{
			`CREATE EXTENSION IF NOT EXISTS hstore`,
			`CREATE EXTENSION IF NOT EXISTS "uuid-ossp"`,
			`CREATE TABLE apps (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  name varchar(30) NOT NULL,
  github_repo text,
  docker_repo text,
  created_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE TABLE configs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  vars hstore,
  created_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE TABLE slugs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  image text NOT NULL,
  process_types hstore NOT NULL
)`,
			`CREATE TABLE releases (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  config_id uuid NOT NULL references configs(id) ON DELETE CASCADE,
  slug_id uuid NOT NULL references slugs(id) ON DELETE CASCADE,
  version int NOT NULL,
  description text,
  created_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE TABLE processes (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  release_id uuid NOT NULL references releases(id) ON DELETE CASCADE,
  "type" text NOT NULL,
  quantity int NOT NULL,
  command text NOT NULL
)`,
			`CREATE TABLE jobs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  release_version int NOT NULL,
  process_type text NOT NULL,
  instance int NOT NULL,

  environment hstore NOT NULL,
  image text NOT NULL,
  command text NOT NULL,
  updated_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE TABLE deployments (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  release_id uuid references releases(id),
  image text NOT NULL,
  status text NOT NULL,
  error text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  finished_at timestamp without time zone
)`,
			`CREATE UNIQUE INDEX index_apps_on_name ON apps USING btree (name)`,
			`CREATE UNIQUE INDEX index_apps_on_github_repo ON apps USING btree (github_repo)`,
			`CREATE UNIQUE INDEX index_apps_on_docker_repo ON apps USING btree (docker_repo)`,
			`CREATE UNIQUE INDEX index_processes_on_release_id_and_type ON processes USING btree (release_id, "type")`,
			`CREATE UNIQUE INDEX index_slugs_on_image ON slugs USING btree (image)`,
			`CREATE UNIQUE INDEX index_releases_on_app_id_and_version ON releases USING btree (app_id, version)`,
			`CREATE UNIQUE INDEX index_jobs_on_app_id_and_release_version_and_process_type_and_instance ON jobs (app_id, release_version, process_type, instance)`,
			`CREATE INDEX index_configs_on_created_at ON configs (created_at)`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE apps CASCADE`,
			`DROP TABLE configs CASCADE`,
			`DROP TABLE slugs CASCADE`,
			`DROP TABLE releases CASCADE`,
			`DROP TABLE processes CASCADE`,
			`DROP TABLE jobs CASCADE`,
			`DROP TABLE deployments CASCADE`,
		}),
	},
	{
		ID: 2,
		Up: migrate.Queries([]string{
			`CREATE TABLE domains (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  hostname text NOT NULL,
  created_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE INDEX index_domains_on_app_id ON domains USING btree (app_id)`,
			`CREATE UNIQUE INDEX index_domains_on_hostname ON domains USING btree (hostname)`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE domains CASCADE`,
		}),
	},
	{
		ID: 3,
		Up: migrate.Queries([]string{
			`DROP TABLE jobs`,
		}),
		Down: migrate.Queries([]string{
			`CREATE TABLE jobs (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  release_version int NOT NULL,
  process_type text NOT NULL,
  instance int NOT NULL,

  environment hstore NOT NULL,
  image text NOT NULL,
  command text NOT NULL,
  updated_at timestamp without time zone default (now() at time zone 'utc')
)`,
		}),
	},
	{
		ID: 4,
		Up: migrate.Queries([]string{
			`CREATE TABLE ports (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  port integer,
  app_id uuid references apps(id) ON DELETE SET NULL
)`,
			`-- Insert 1000 ports
INSERT INTO ports (port) (SELECT generate_series(9000,10000))`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE ports CASCADE`,
		}),
	},
	{
		ID: 5,
		Up: migrate.Queries([]string{
			`ALTER TABLE apps DROP COLUMN docker_repo`,
			`ALTER TABLE apps DROP COLUMN github_repo`,
			`ALTER TABLE apps ADD COLUMN repo text`,
			`DROP TABLE deployments`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE apps DROP COLUMN repo`,
			`ALTER TABLE apps ADD COLUMN docker_repo text`,
			`ALTER TABLE apps ADD COLUMN github_repo text`,
			`CREATE TABLE deployments (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id text NOT NULL references apps(name) ON DELETE CASCADE,
  release_id uuid references releases(id),
  image text NOT NULL,
  status text NOT NULL,
  error text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  finished_at timestamp without time zone
)`,
		}),
	},
	{
		ID: 6,
		Up: migrate.Queries([]string{
			`DROP INDEX index_slugs_on_image`,
		}),
		Down: migrate.Queries([]string{
			`CREATE UNIQUE INDEX index_slugs_on_image ON slugs USING btree (image)`,
		}),
	},
	{
		ID: 7,
		Up: migrate.Queries([]string{
			`-- Values: private, public
ALTER TABLE apps ADD COLUMN exposure TEXT NOT NULL default 'private'`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE apps DROP COLUMN exposure`,
		}),
	},
	{
		ID: 8,
		Up: migrate.Queries([]string{
			`CREATE TABLE certificates (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  app_id uuid NOT NULL references apps(id) ON DELETE CASCADE,
  name text,
  certificate_chain text,
  created_at timestamp without time zone default (now() at time zone 'utc'),
  updated_at timestamp without time zone default (now() at time zone 'utc')
)`,
			`CREATE UNIQUE INDEX index_certificates_on_app_id ON certificates USING btree (app_id)`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE certificates CASCADE`,
		}),
	},
	{
		ID: 9,
		Up: migrate.Queries([]string{
			`ALTER TABLE processes ADD COLUMN cpu_share int`,
			`ALTER TABLE processes ADD COLUMN memory int`,
			`UPDATE processes SET cpu_share = 256, memory = 1073741824`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE processes DROP COLUMN cpu_share`,
			`ALTER TABLE processes DROP COLUMN memory`,
		}),
	},
	{
		ID: 10,
		Up: migrate.Queries([]string{
			`ALTER TABLE processes ALTER COLUMN memory TYPE bigint`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE processes ALTER COLUMN memory TYPE integer`,
		}),
	},
	{
		ID: 11,
		Up: migrate.Queries([]string{
			`ALTER TABLE apps ADD COLUMN cert text`,
			`UPDATE apps SET cert = (select name from certificates where certificates.app_id = apps.id)`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE apps DROP COLUMN cert`,
		}),
	},
	{
		ID: 12,
		Up: migrate.Queries([]string{
			`ALTER TABLE processes ADD COLUMN nproc bigint`,
			`UPDATE processes SET nproc = 0`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE processes DROP COLUMN nproc`,
		}),
	},
	{
		ID: 13,
		Up: migrate.Queries([]string{
			`ALTER TABLE ports ADD COLUMN taken text`,
			`UPDATE ports SET taken = 't' FROM (SELECT port FROM ports WHERE app_id is not NULL) as used_ports WHERE ports.port = used_ports.port`,
		}),
		Down: migrate.Queries([]string{
			`ALTER TABLE ports DROP column taken`,
		}),
	},

	// This migration changes how we store commands from a plain string to a
	// []string.
	{
		ID: 14,
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE slugs ADD COLUMN process_types_json json`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE processes ADD COLUMN command_json json`)
			if err != nil {
				return err
			}

			// Migrate the data in the slugs table
			rows, err := tx.Query(`SELECT id, process_types FROM slugs`)
			if err != nil {
				return err
			}

			slugs := make(map[string]map[string]Command)
			for rows.Next() {
				var id string
				var ptypes hstore.Hstore
				if err := rows.Scan(&id, &ptypes); err != nil {
					return err
				}
				m := make(map[string]Command)
				for k, v := range ptypes.Map {
					command, err := ParseCommand(v.String)
					if err != nil {
						return err
					}
					m[k] = command
				}
				slugs[id] = m
			}

			if err := rows.Err(); err != nil {
				return err
			}

			rows.Close()

			for id, ptypes := range slugs {
				raw, err := json.Marshal(ptypes)
				if err != nil {
					return err
				}

				_, err = tx.Exec(`UPDATE slugs SET process_types_json = $1 WHERE id = $2`, raw, id)
				if err != nil {
					return err
				}
			}

			_, err = tx.Exec(`ALTER TABLE slugs DROP COLUMN process_types`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE slugs RENAME COLUMN process_types_json to process_types`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE slugs ALTER COLUMN process_types SET NOT NULL`)
			if err != nil {
				return err
			}

			// Migrate the data in the processes table.
			rows, err = tx.Query(`SELECT id, command FROM processes`)
			if err != nil {
				return err
			}

			commands := make(map[string]string)
			for rows.Next() {
				var id, command string
				if err := rows.Scan(&id, &command); err != nil {
					return err
				}
				commands[id] = command
			}

			if err := rows.Err(); err != nil {
				return err
			}

			rows.Close()

			for id, command := range commands {
				cmd, err := ParseCommand(command)
				if err != nil {
					return err
				}

				query := `UPDATE processes SET command_json = $1 WHERE id = $2`
				_, err = tx.Exec(query, cmd, id)
				if err != nil {
					return err
				}
			}

			_, err = tx.Exec(`ALTER TABLE processes DROP COLUMN command`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE processes RENAME COLUMN command_json TO command`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE processes ALTER COLUMN command SET NOT NULL`)
			return err
		},
		Down: migrate.Queries([]string{
			`ALTER TABLE processes DROP COLUMN command`,
			`ALTER TABLE processes ADD COLUMN command text not null`,
			`ALTER TABLE slugs DROP COLUMN process_types`,
			`ALTER TABLE slugs ADD COLUMN process_types hstore not null`,
		}),
	},

	// This migration changes that way we store process configuration for
	// releases and slugs, to instead store a Formation in JSON format.
	{
		ID: 15,
		Up: func(tx *sql.Tx) error {
			_, err := tx.Exec(`ALTER TABLE slugs ADD COLUMN procfile bytea`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE releases ADD COLUMN formation json`)
			if err != nil {
				return err
			}

			rows, err := tx.Query(`SELECT id, process_types FROM slugs`)
			if err != nil {
				return err
			}

			slugs := make(map[string]procfile.Procfile)
			for rows.Next() {
				var id string
				var ptypes []byte
				if err := rows.Scan(&id, &ptypes); err != nil {
					return err
				}
				m := make(map[string][]string)
				if err := json.Unmarshal(ptypes, &m); err != nil {
					return err
				}
				p := make(procfile.ExtendedProcfile)
				for name, command := range m {
					p[name] = procfile.Process{
						Command: command,
					}
				}
				slugs[id] = p
			}

			if err := rows.Err(); err != nil {
				return err
			}

			rows.Close()

			for id, p := range slugs {
				raw, err := procfile.Marshal(p)
				if err != nil {
					return err
				}

				_, err = tx.Exec(`UPDATE slugs SET procfile = $1 WHERE id = $2`, raw, id)
				if err != nil {
					return err
				}
			}

			_, err = tx.Exec(`ALTER TABLE slugs DROP COLUMN process_types`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`ALTER TABLE slugs ALTER COLUMN procfile SET NOT NULL`)
			if err != nil {
				return err
			}

			rows, err = tx.Query(`SELECT release_id, id, type, quantity, command, memory, cpu_share, nproc FROM processes`)
			if err != nil {
				return err
			}

			formations := make(map[string]Formation)
			for rows.Next() {
				var release, id, ptype string
				var command Command
				var quantity, memory, cpu, nproc int
				if err := rows.Scan(&release, &id, &ptype, &quantity, &command, &memory, &cpu, &nproc); err != nil {
					return err
				}
				if formations[release] == nil {
					formations[release] = make(Formation)
				}

				f := formations[release]
				f[ptype] = Process{
					Command:  command,
					Quantity: quantity,
					Memory:   constraints.Memory(memory),
					CPUShare: constraints.CPUShare(cpu),
					Nproc:    constraints.Nproc(nproc),
				}
			}

			if err := rows.Err(); err != nil {
				return err
			}

			rows.Close()

			for id, f := range formations {
				_, err = tx.Exec(`UPDATE releases SET formation = $1 WHERE id = $2`, f, id)
				if err != nil {
					return err
				}
			}

			_, err = tx.Exec(`ALTER TABLE releases ALTER COLUMN formation SET NOT NULL`)
			if err != nil {
				return err
			}

			_, err = tx.Exec(`DROP TABLE processes`)

			return err
		},
		Down: migrate.Queries([]string{
			`ALTER TABLE releases DROP COLUMN formation`,
			`ALTER TABLE slugs DROP COLUMN procfile`,
			`ALTER TABLE slugs ADD COLUMN process_types hstore not null`,
			`CREATE TABLE processes (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  release_id uuid NOT NULL references releases(id) ON DELETE CASCADE,
  "type" text NOT NULL,
  quantity int NOT NULL,
  command text NOT NULL,
  cpu_share int,
  memory bigint,
  nproc bigint
)`,
			`CREATE UNIQUE INDEX index_processes_on_release_id_and_type ON processes USING btree (release_id, "type")`,
		}),
	},

	// This migration changes that way we store process configuration for
	// releases and slugs, to instead store a Formation in JSON format.
	{
		ID: 16,
		Up: migrate.Queries([]string{
			`CREATE TABLE stacks (
  app_id text NOT NULL,
  stack_name text NOT NULL
)`,
			`CREATE UNIQUE INDEX index_stacks_on_app_id ON stacks USING btree (app_id)`,
			`CREATE UNIQUE INDEX index_stacks_on_stack_name ON stacks USING btree (stack_name)`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE stacks`,
		}),
	},

	// This migration adds a table that gets used to migrate apps from the
	// old ECS backend to the shiny new CloudFormation backend.
	{
		ID: 17,
		Up: migrate.Queries([]string{
			`CREATE TABLE scheduler_migration (app_id text NOT NULL, backend text NOT NULL)`,
			`INSERT INTO scheduler_migration (app_id, backend) SELECT id, 'ecs' FROM apps`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE scheduler_migration`,
		}),
	},

	// This migration adds a table that stores environment variables for the
	// Custom::ECSEnvironment resource.
	{
		ID: 18,
		Up: migrate.Queries([]string{
			`CREATE TABLE ecs_environment (
  id uuid NOT NULL DEFAULT uuid_generate_v4() primary key,
  environment json NOT NULL
)`,
		}),
		Down: migrate.Queries([]string{
			`DROP TABLE ecs_environment`,
		}),
	},
}

// latestSchema returns the schema version that this version of Empire should be
// using.
func latestSchema() int {
	return migrations[len(migrations)-1].ID
}
