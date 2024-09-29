package main

import (
	prom "github.com/hasura/ndc-prometheus/connector"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/connector"
)

// Start the connector server at http://localhost:8080
//
//	go run . serve
//
// See [NDC Go SDK] for more information.
//
// [NDC Go SDK]: https://github.com/hasura/ndc-sdk-go
func main() {
	if err := connector.Start[metadata.Configuration, metadata.State](
		prom.NewPrometheusConnector(),
		connector.WithMetricsPrefix("ndc_prometheus"),
		connector.WithDefaultServiceName("ndc_prometheus"),
	); err != nil {
		panic(err)
	}
}
