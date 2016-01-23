// Package migrations contains migrations for migrating the database that Empire
// uses.
package migrations

import (
	"database/sql"

	"github.com/rubenv/sql-migrate"
)

// Migrates a sql.DB up.
func Up(db *sql.DB, dialect string) error {
	if _, err := migrate.Exec(db, dialect, SchemaMigration, migrate.Up); err != nil {
		return err
	}

	_, err := migrate.Exec(db, dialect, Migrations, migrate.Up)
	return err
}

// Migrates a sql.DB down.
func Down(db *sql.DB, dialect string) error {
	_, err := migrate.Exec(db, dialect, Migrations, migrate.Down)
	return err
}

// Migrations contains the database migrations to reach the final schema.
var Migrations = &migrate.AssetMigrationSource{
	Asset:    Asset,
	AssetDir: AssetDir,
	Dir:      "migrations",
}

// SchemaMigration migrates the migration schema from the
// github.com/mattes/migrate format to the github.com/rubenv/sql-migrate format.
//
// Note that this has to run as a separate call to migrate.Exec since the
// migration plan will be cached.
var SchemaMigration = &migrate.MemoryMigrationSource{
	Migrations: []*migrate.Migration{
		&migrate.Migration{
			Id: "schema_migrations_to_gorp_migrations",
			Up: []string{
				"CREATE TABLE IF NOT EXISTS schema_migrations (version integer not null);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0001_initial_schema.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 1);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0002_add_domains.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 2);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0003_remove_jobs.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 3);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0004_add_ports.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 4);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0005_add_repo.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 5);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0006_remove_unique_constraint_on_image.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 6);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0007_add_app_exposure.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 7);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0008_add_certificates.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 8);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0009_add_constraints_column.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 9);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0010_memory_bigint.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 10);",
				"INSERT into gorp_migrations (id, applied_at) SELECT '0011_move_certs.sql', now() WHERE EXISTS (SELECT version from schema_migrations where version = 11);",
				"DROP TABLE schema_migrations;",
			},
			Down: []string{},
		},
	},
}
