package db_test

import (
	"context"
	"path/filepath"
	"testing"

	"github.com/jsnfwlr/go11y"
	"github.com/jsnfwlr/go11y/db"
	"github.com/jsnfwlr/go11y/etc/migrations"
	"github.com/jsnfwlr/go11y/testingContainers"
)

func TestFileSystem(t *testing.T) {
	fs := migrations.Filesystem
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

func TesDatabase(t *testing.T) {
	col, err := migrations.New()
	if err != nil {
		t.Fatalf("could not create the embedded filesystem: %v", err)
	}

	ctx := context.Background()

	ctr, err := testingContainers.Postgres(t, ctx, "17")
	if err != nil {
		t.Fatalf("could not start the Postgres container: %v", err)
	}

	defer ctr.Cleanup(t)

	gCfg, err := go11y.LoadConfig()
	if err != nil {
		t.Fatalf("could not load the configuration: %v", err)
	}

	ctx, o, err := go11y.Initialise(ctx, gCfg, nil)
	if err != nil {
		t.Fatalf("could not initialise the observer: %v", err)
	}

	t.Run("migrate", func(t *testing.T) {
		m, err := db.NewMigrator(ctx, o, ctr, col)
		if err != nil {
			t.Fatalf("could not create the migrator: %v", err)
		}

		i, err := m.Info(-1)
		if err != nil {
			t.Fatalf("could not get the migration info: %v", err)
		}

		t.Logf("port: %s, database: %s, currentVersion: %d, targetVersion: %d\n%s", i.Port, i.Database, i.Migrations.CurrentVersion, i.Migrations.TargetVersion, i.Migrations.Summary)

		err = db.RunMigrations(ctx, o, ctr, col, -1, true)
		if err != nil {
			t.Fatalf("could not run the migrations: %v", err)
		}
	})
}
