package connector

import (
	"log/slog"
	"testing"

	"github.com/hasura/ndc-sdk-go/ndctest"
)

func TestConnector(t *testing.T) {
	t.Setenv("CONNECTION_URL", "http://localhost:9090")
	t.Setenv("PROMETHEUS_USERNAME", "admin")
	t.Setenv("PROMETHEUS_PASSWORD", "test")
	slog.SetLogLoggerLevel(slog.LevelError)
	ndctest.TestConnector(t, NewPrometheusConnector(), ndctest.TestConnectorOptions{
		Configuration: "../tests/configuration",
		TestDataDir:   "testdata",
	})
}
