// The Hasura Prometheus Connector allows for connecting to a Prometheus database
// giving you an instant GraphQL API on top of your Prometheus data.
package main

import (
	prom "github.com/hasura/ndc-prometheus/connector"
	"github.com/hasura/ndc-sdk-go/connector"
)

func main() {
	if err := connector.Start(
		prom.NewPrometheusConnector(),
		connector.WithMetricsPrefix("ndc_prometheus"),
		connector.WithDefaultServiceName("ndc_prometheus"),
	); err != nil {
		panic(err)
	}
}
