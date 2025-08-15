package go11y

import (
	"context"
	"testing"

	_ "github.com/jackc/pgx/v5/stdlib"
	"github.com/testcontainers/testcontainers-go"
	grafanalgtm "github.com/testcontainers/testcontainers-go/modules/grafana-lgtm"
	"github.com/testcontainers/testcontainers-go/modules/postgres"
)

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

	dConStr, err := ctr.ConnectionString(ctx, "sslmode=disable")
	if err != nil {
		t.Fatalf("failed to get connection string: %v", err)
		return nil, err
	}

	t.Setenv("DB_CONSTR", dConStr)

	return c, nil
}

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
