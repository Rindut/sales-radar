package store

import (
	"database/sql"
	"os"
	"path/filepath"
	"testing"
)

func TestParseMigrationVersion(t *testing.T) {
	v, err := parseMigrationVersion("001_init.sql")
	if err != nil || v != 1 {
		t.Fatalf("parseMigrationVersion = (%d, %v), want (1, nil)", v, err)
	}
	if _, err := parseMigrationVersion("badname.sql"); err == nil {
		t.Fatal("expected error for non-numeric migration prefix")
	}
}

func TestRunSQLMigrations_AppliesAndRecords(t *testing.T) {
	tmp := t.TempDir()
	migDir := filepath.Join(tmp, "migrations")
	if err := os.MkdirAll(migDir, 0o755); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "001_first.sql"), []byte(`
CREATE TABLE IF NOT EXISTS t_first (id INTEGER PRIMARY KEY);
`), 0o644); err != nil {
		t.Fatal(err)
	}
	if err := os.WriteFile(filepath.Join(migDir, "002_second.sql"), []byte(`
CREATE TABLE IF NOT EXISTS t_second (id INTEGER PRIMARY KEY);
`), 0o644); err != nil {
		t.Fatal(err)
	}
	db, err := sql.Open("sqlite", ":memory:")
	if err != nil {
		t.Fatal(err)
	}
	defer db.Close()
	if err := runSQLMigrations(db, migDir); err != nil {
		t.Fatal(err)
	}
	var n int
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("schema_migrations count = %d, want 2", n)
	}
	// idempotent second run
	if err := runSQLMigrations(db, migDir); err != nil {
		t.Fatal(err)
	}
	if err := db.QueryRow(`SELECT COUNT(*) FROM schema_migrations`).Scan(&n); err != nil {
		t.Fatal(err)
	}
	if n != 2 {
		t.Fatalf("schema_migrations count after rerun = %d, want 2", n)
	}
}

