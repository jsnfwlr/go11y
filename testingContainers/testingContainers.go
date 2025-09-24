package testingContainers

import (
	"context"
	"fmt"
	"testing"

	"github.com/docker/go-connections/nat"
	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	grafanalgtm "github.com/testcontainers/testcontainers-go/modules/grafana-lgtm"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

type DatabaseContainer struct {
	Postgres *postgres.PostgresContainer
	Database string
	Username string
	Password string
	Host     string
	Port     string
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

func (c DatabaseContainer) Cleanup(t testing.TB) {
	if c.Postgres == nil {
		testcontainers.CleanupContainer(t, c.Postgres)
	}
}

func (c DatabaseContainer) MappedPort(t testing.TB, ctx context.Context, port string) string {
	t.Helper()
	mappedPort, err := c.Postgres.MappedPort(ctx, nat.Port(port))
	if err != nil {
		t.Fatalf("could not get mapped port %s: %v", port, err)
	}

	return mappedPort.Port()
}

func (c DatabaseContainer) Hostname(t testing.TB, ctx context.Context) string {
	t.Helper()
	host, err := c.Postgres.Host(ctx)
	if err != nil {
		t.Fatalf("could not get host: %v", err)
	}

	return host
}

func (c DatabaseContainer) DBConStr() string {
	return fmt.Sprintf("postgres://%s:%s@%s:%s/%s?sslmode=disable", c.Username, c.Password, c.Host, c.Port, c.Database)
}

/*
func Postgres(t *testing.T, ctx context.Context) (ctr *postgres.PostgresContainer, fault error) {
	// This function is a placeholder for starting the OpenTelemetry server.
	// In a real test, you would start the server here and ensure it's running.
	t.Helper()
	t.Log("Starting Postgres container for testing...")

	c, err := postgres.Run(
		ctx,
		"postgres:17",
		postgres.WithDatabase("api_calls"),
		postgres.WithUsername("user"),
		postgres.WithPassword("password"),

		postgres.BasicWaitStrategies(),
		postgres.WithSQLDriver("pgx"),
		testcontainers.WithExposedPorts("8642"),
	)
	if err != nil {
		t.Errorf("failed to start Postgres container: %s", err)
		return nil, err
	}

	dConStr, err := c.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
		return nil, err
	}

	t.Setenv("DB_CONSTR", dConStr)

	return c, nil
}
*/

func LGTM(t *testing.T, ctx context.Context) (ctr *grafanalgtm.GrafanaLGTMContainer, fault error) {
	t.Helper()
	t.Log("Starting Grafana LGTM container for testing...")

	c, err := grafanalgtm.Run(
		ctx,
		"grafana/otel-lgtm:0.6.0",
		testcontainers.WithExposedPorts("8318/tcp", "8317/tcp"),
		grafanalgtm.WithAdminCredentials("admin", "admin"),
	)
	if err != nil {
		t.Errorf("failed to start Grafana LGTM container: %s", err)
		return nil, err
	}

	host, err := c.Host(ctx)
	if err != nil {
		t.Fatalf("failed to get container host: %v", err)
	}

	port, err := c.MappedPort(ctx, "4318/tcp")
	if err != nil {
		t.Fatalf("failed to get mapped port: %v", err)
	}

	t.Setenv("OTEL_HOST", host)
	t.Setenv("OTEL_PORT", port.Port())

	t.Logf("Grafana LGTM is running at %s:%s", host, port.Port())

	return c, nil
}
