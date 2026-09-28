package main

import "testing"

func TestMigrationDatabaseURL(t *testing.T) {
	t.Setenv("DATABASE_URL", "runtime")
	t.Setenv("MIGRATION_DATABASE_URL", "migration")
	if got := migrationDatabaseURL(); got != "migration" {
		t.Fatalf("migration database URL = %q, want migration", got)
	}

	t.Setenv("MIGRATION_DATABASE_URL", "")
	if got := migrationDatabaseURL(); got != "runtime" {
		t.Fatalf("fallback database URL = %q, want runtime", got)
	}
}
