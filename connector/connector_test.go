package connector

import (
	"log/slog"
	"testing"

	"github.com/hasura/ndc-sdk-go/ndctest"
)

func TestConnector(t *testing.T) {
	setDefaultEnvironments(t)
	slog.SetLogLoggerLevel(slog.LevelError)
	ndctest.TestConnector(t, NewPrometheusConnector(), ndctest.TestConnectorOptions{
		Configuration: "../tests/configuration",
		TestDataDir:   "testdata",
	})
}

func TestConnectorPromptQL(t *testing.T) {
	setDefaultEnvironments(t)
	slog.SetLogLoggerLevel(slog.LevelError)
	ndctest.TestConnector(t, NewPrometheusConnector(), ndctest.TestConnectorOptions{
		Configuration: "../tests/configuration-promptql",
		TestDataDir:   "testdata/empty",
	})
}

func setDefaultEnvironments(t *testing.T) {
	t.Setenv("CONNECTION_URL", "http://localhost:9090")
	t.Setenv("PROMETHEUS_USERNAME", "admin")
	t.Setenv("PROMETHEUS_PASSWORD", "test")
}
