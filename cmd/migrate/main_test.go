package main

import (
	"net/url"
	"testing"
)

func TestMigrationURLDisablesPreparedStatementCache(t *testing.T) {
	got, err := migrationURL("postgresql://user:p%40ss@db.example:6543/postgres?sslmode=require")
	if err != nil {
		t.Fatalf("migrationURL returned an error: %v", err)
	}

	u, err := url.Parse(got)
	if err != nil {
		t.Fatalf("parse migration URL: %v", err)
	}
	if u.Scheme != "pgx5" {
		t.Fatalf("scheme = %q, want pgx5", u.Scheme)
	}
	if got := u.Query().Get("sslmode"); got != "require" {
		t.Fatalf("sslmode = %q, want require", got)
	}
	if got := u.Query().Get("default_query_exec_mode"); got != "exec" {
		t.Fatalf("default_query_exec_mode = %q, want exec", got)
	}
	if password, _ := u.User.Password(); password != "p@ss" {
		t.Fatalf("password = %q, want p@ss", password)
	}
}

func TestMigrationURLRejectsUnsupportedScheme(t *testing.T) {
	if _, err := migrationURL("mysql://localhost/database"); err == nil {
		t.Fatal("migrationURL accepted an unsupported scheme")
	}
}
