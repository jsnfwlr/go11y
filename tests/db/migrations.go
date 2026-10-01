// Package db provides database migration functionality using embedded filesystem for migration files.
package db

import (
	"context"
	"embed"
	"fmt"
	"io/fs"
	"os"

	"github.com/jackc/pgx/v5"
	migrate "github.com/jackc/tern/v2/migrate"
)

// MigrationFS provides methods to interact with an embedded filesystem for migrations.
type MigrationFS struct {
	FS embed.FS
}

// ReadDir reads the directory from the embedded filesystem.
func (m MigrationFS) ReadDir(name string) ([]fs.FileInfo, error) {
	files, err := m.FS.ReadDir(name)
	if err != nil {
		return nil, fmt.Errorf("could not get the files from the embedded filesystem: %w", err)
	}

	var r []os.FileInfo

	for _, f := range files {
		fi, err := f.Info()
		if err != nil {
			return nil, fmt.Errorf("could not get information for file '%s': %w", f.Name(), err)
		}

		r = append(r, fi)
	}

	return r, nil
}

// ReadFile reads a file from the embedded filesystem.
func (m MigrationFS) ReadFile(name string) (contents []byte, fault error) {
	b, err := m.FS.ReadFile(name)
	if err != nil {
		return nil, fmt.Errorf("could not read file '%s' from embedded filesystem: %w", name, err)
	}

	return b, nil
}

// Glob returns the file paths matching the given pattern from the embedded filesystem.
func (m MigrationFS) Glob(pattern string) (matches []string, fault error) {
	matches, err := fs.Glob(m.FS, pattern)
	if err != nil {
		return nil, fmt.Errorf("could not get glob matches for pattern '%s': %w", pattern, err)
	}

	return matches, nil
}

// Open opens a file from the embedded filesystem.
func (m MigrationFS) Open(name string) (file fs.File, fault error) {
	f, err := m.FS.Open(name)
	if err != nil {
		return nil, fmt.Errorf("could not open file '%s' from embedded filesystem: %w", name, err)
	}

	return f, nil
}

// Configurator defines the interface for database configuration.
type Configurator interface {
	DatabaseURL() string
}

// DBMigrator handles database migrations.
type DBMigrator struct {
	context       context.Context
	connection    *pgx.Conn
	migrator      *migrate.Migrator
	configuration Configurator
	logger        Logger
}

// FilesystemProvider defines the interface for providing migration files from a filesystem.
type FilesystemProvider interface {
	ReadDir(name string) ([]fs.FileInfo, error)
	ReadFile(name string) ([]byte, error)
	Open(name string) (fs.File, error)
}

// NewMigrator creates a new DBMigrator instance.
func NewMigrator(ctx context.Context, logger Logger, connParams Configurator, fs FilesystemProvider) (migrator DBMigrator, fault error) {
	conn, err := pgx.Connect(ctx, connParams.DatabaseURL())
	if err != nil {
		return DBMigrator{}, fmt.Errorf("could not connect to database: %w", err)
	}

	mo := &migrate.MigratorOptions{
		DisableTx: false,
	}

	mig, err := migrate.NewMigratorEx(ctx, conn, "db_version", mo)
	if err != nil {
		return DBMigrator{}, fmt.Errorf("could not create migratorEx %w", err)
	}

	err = mig.LoadMigrations(fs)
	if err != nil {
		return DBMigrator{}, fmt.Errorf("could not load migrations: %w", err)
	}

	return DBMigrator{
		context:       ctx,
		connection:    conn,
		migrator:      mig,
		configuration: connParams,
		logger:        logger,
	}, nil
}

// Info holds information about the current migration status.
type Info struct {
	DBConnStr  string
	Port       string
	Database   string
	Migrations MigrationInfo
}

// MigrationInfo holds information about the migration status.
type MigrationInfo struct {
	CurrentVersion int32
	TargetVersion  int32
	Stages         []Stage
	Summary        string
}

// Stage represents a single migration stage.
type Stage struct {
	Sequence int32
	Name     string
	Migrated bool
}

// ErrInvalidSequenceNumber returns an error indicating an invalid sequence number.
func ErrInvalidSequenceNumber(seq int32) error {
	return fmt.Errorf("provided value '%d' is an invalid sequence number", seq)
}

// GetCurrentVersion retrieves the current migration version from the database.
func (m DBMigrator) GetCurrentVersion() (currentVersion int32, fault error) {
	return m.migrator.GetCurrentVersion(m.context)
}

// Info provides information about the current migration status.
func (m DBMigrator) Info(stopAfter int32) (information Info, fault error) {
	var err error

	i := Info{
		DBConnStr:  m.configuration.DatabaseURL(),
		Migrations: MigrationInfo{},
	}

	i.Migrations.CurrentVersion, err = m.migrator.GetCurrentVersion(m.context)
	if err != nil {
		return Info{}, fmt.Errorf("could not get current version: %w", err)
	}

	if stopAfter < 0 {
		stopAfter = m.migrator.Migrations[len(m.migrator.Migrations)-1].Sequence
	}

	for _, mig := range m.migrator.Migrations {
		// i.Migrations.Stages = append(i.Migrations.Stages, mig.Sequence)
		ind := "  "

		s := Stage{
			Sequence: mig.Sequence,
			Name:     mig.Name,
			Migrated: mig.Sequence <= i.Migrations.CurrentVersion,
		}
		i.Migrations.Stages = append(i.Migrations.Stages, s)

		if mig.Sequence == stopAfter {
			ind = "> "
		}

		if mig.Sequence == i.Migrations.CurrentVersion {
			ind = "@ "
		}

		i.Migrations.Summary += fmt.Sprintf("%2s %3d %s\n", ind, mig.Sequence, mig.Name)
	}

	return i, nil
}

// Migrate migrates the database to the latest version.
func (m *DBMigrator) Migrate() (fault error) {
	m.migrator.OnStart = func(sequence int32, name string, direction string, sql string) {
		if direction == "up" {
			fmt.Printf("Migrating %d: %s\n", sequence, name)
		} else {
			fmt.Printf("Rolling back %d: %s\n", sequence, name)
		}
	}

	err := m.migrator.Migrate(m.context)
	if err != nil {
		return fmt.Errorf("could not migrate: %w", err)
	}

	return nil
}

// MigrateTo migrates the database to the specified sequence number.
func (m *DBMigrator) MigrateTo(sequence int32) (fault error) {
	m.migrator.OnStart = func(sequence int32, name string, direction string, _ string) {
		// if direction == "up" {
		// } else {
		// 	fmt.Printf("Rolling back %d: %s\n", sequence, name)
		// }

		fmt.Printf("%s-grading %s (v%d)\n", direction, name, sequence)
	}

	err := m.migrator.MigrateTo(m.context, sequence)
	if err != nil {
		return fmt.Errorf("could not migrate to %d: %w", sequence, err)
	}

	return nil
}

// RunMigrations runs the database migrations to the specified version.
func RunMigrations(ctx context.Context, logger Logger, connParams Configurator, fs FilesystemProvider, stopAfter int32, printSummary bool) (fault error) {
	m, err := NewMigrator(ctx, logger, connParams, fs)
	if err != nil {
		return fmt.Errorf("could not create migrator: %w", err)
	}

	info, err := m.Info(stopAfter)
	if err != nil {
		return fmt.Errorf("could not get migration info: %w", err)
	}

	if printSummary {
		fmt.Printf("Migrations for %s:%s/%s\n", info.DBConnStr, info.Port, info.Database)
		fmt.Println(info.Migrations.Summary)
	}

	if stopAfter >= 0 {
		direction := "upgrade"
		if info.Migrations.CurrentVersion > stopAfter {
			direction = "downgrade"
		}

		fmt.Printf("Starting %s from v%d to v%d\n", direction, info.Migrations.CurrentVersion, stopAfter)

		err = m.MigrateTo(stopAfter)
		if err != nil {
			return fmt.Errorf("could not complete %s from v%d to v%d: %w", direction, info.Migrations.CurrentVersion, stopAfter, err)
		}

		return nil
	}

	err = m.Migrate()
	if err != nil {
		return fmt.Errorf("could not migrate: %w", err)
	}

	return nil
}

// Logger defines the logging interface used by the DBMigrator and allows go11y to be used to log messages.
type Logger interface {
	Debug(msg string, ephemeralArgs ...any)
	Info(msg string, ephemeralArgs ...any)
	Error(msg string, err error, severity string, ephemeralArgs ...any)
}
