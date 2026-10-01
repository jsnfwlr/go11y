package containers

import (
	"context"
	"testing"

	"github.com/testcontainers/testcontainers-go"
	grafanalgtm "github.com/testcontainers/testcontainers-go/modules/grafana-lgtm"
)

// LGTM starts a Grafana LGTM container for testing purposes.
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
