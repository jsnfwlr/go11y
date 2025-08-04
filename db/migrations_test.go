package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jsnfwlr/o11y"
	"github.com/jsnfwlr/o11y/config"
	"github.com/jsnfwlr/o11y/db"
	"github.com/jsnfwlr/o11y/etc/migrations"
	"github.com/testcontainers/testcontainers-go"
)

func TestFileSystem(t *testing.T) {
	fs := migrations.Migrations
	em := db.MigrationFS{FS: fs}

	fi, err := em.ReadDir(".")
	if err != nil {
		t.Fatalf("could not read the directory: %v", err)
	}

	if len(fi) == 0 {
		t.Fatalf("no files found in the directory")
	}

	for _, f := range fi {
		t.Logf("name: %s, size: %d, mode: %s, modTime: %v, isDir: %t", f.Name(), f.Size(), f.Mode(), f.ModTime(), f.IsDir())
	}

	sharedPaths, err := em.Glob(filepath.Join("*", "*.sql"))
	if err != nil {
		t.Errorf("could not get globs: %s", err)
	}

	for _, p := range sharedPaths {
		t.Logf("path: %s", p)
	}
}

func TestMigrator(t *testing.T) {
	fs := migrations.Migrations
	ctx := context.Background()

	ctr, err := o11y.Postgres(t, ctx)
	if err != nil {
		t.Fatalf("could not start Postgres container: %v", err)
	}

	testcontainers.CleanupContainer(t, ctr)

	dConStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("could not get the connection string: %v", err)
	}
	t.Setenv("DB_CONSTR", dConStr)

	cfg, err := config.Load()
	if err != nil {
		t.Fatalf("could not load the configuration: %v", err)
	}

	o := o11y.Get(ctx)

	m, err := db.NewMigrator(ctx, o, cfg, fs)
	if err != nil {
		t.Fatalf("could not create the migrator: %v", err)
	}

	i, err := m.Info(-1)
	if err != nil {
		t.Fatalf("could not get the migration info: %v", err)
	}

	t.Logf("host: %s, currentVersion: %d, targetVersion: %d\n%s", i.DBConnStr, i.Migrations.CurrentVersion, i.Migrations.TargetVersion, i.Migrations.Summary)

	err = db.RunMigrations(ctx, o, cfg, fs, -1, true)
	if err != nil {
		t.Fatalf("could not run the migrations: %v", err)
	}
}
