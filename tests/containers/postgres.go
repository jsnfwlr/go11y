// Package containers provides utilities for setting up testing containers.
// Currently, it includes functionality for
// - Postgres containers
// - Grafana LGTM containers
package containers

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

// DatabaseContainer wraps the PostgresContainer with additional metadata.
type DatabaseContainer struct {
	Postgres *postgres.PostgresContainer
	Database string
	Username string
	Password string
	Host     string
	Port     string
}

// Cleanup terminates the Postgres container.
func (c DatabaseContainer) Cleanup(t testing.TB) {
	if c.Postgres != nil {
		testcontainers.CleanupContainer(t, c.Postgres)
	}
}

// MappedPort returns the mapped port for the given container port.
func (c DatabaseContainer) MappedPort(t testing.TB, ctx context.Context, port string) string {
	t.Helper()
	mappedPort, err := c.Postgres.MappedPort(ctx, nat.Port(port).Port())
	if err != nil {
		t.Fatalf("could not get mapped port %s: %v", port, err)
	}

	return mappedPort.Port()
}

// Hostname returns the hostname of the Postgres container.
func (c DatabaseContainer) Hostname(t testing.TB, ctx context.Context) string {
	t.Helper()
	host, err := c.Postgres.Host(ctx)
	if err != nil {
		t.Fatalf("could not get host: %v", err)
	}

	return host
}

// DatabaseURL returns the connection URL for the Postgres database.
func (c DatabaseContainer) DatabaseURL() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.Database)
}

// Postgres starts a Postgres container for testing purposes.
func Postgres(t *testing.T, ctx context.Context, version string) (container DatabaseContainer, fault error) {
	t.Helper()
	t.Log("Starting Postgres container for testing...")

	var err error

	dbContainer := DatabaseContainer{
		Database: "api_calls",
		Username: "user",
		Password: "password",
	}

	name := fmt.Sprintf("vexil-test-postgres-%s", version)

	dbContainer.Postgres, err = postgres.Run(
		ctx,
		fmt.Sprintf("postgres:%s", version),
		postgres.WithDatabase(dbContainer.Database),
		postgres.WithUsername(dbContainer.Username),
		postgres.WithPassword(dbContainer.Password),
		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
		testcontainers.WithName(name),
		testcontainers.WithReuseByName(name),
	)
	if err != nil {
		t.Errorf("failed to start Postgres container: %s", err)
		return DatabaseContainer{}, err
	}

	dbContainer.Host = dbContainer.Hostname(t, ctx)
	dbContainer.Port = dbContainer.MappedPort(t, ctx, "5432")

	return dbContainer, nil
}
