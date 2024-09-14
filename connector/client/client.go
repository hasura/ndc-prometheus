package client

import (
	"context"
	"net/http"

	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/config"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Client extends the Prometheus API client with advanced methods for the Prometheus connector
type Client struct {
	client api.Client
	tracer trace.Tracer

	v1.API
}

// NewClient creates a new Prometheus client instance
func NewClient(endpoint string, clientConfig config.HTTPClientConfig, tracer trace.Tracer) (*Client, error) {
	httpClient, err := config.NewClientFromConfig(clientConfig, "ndc-prometheus")
	if err != nil {
		return nil, err
	}

	apiClient, err := api.NewClient(api.Config{
		Address: endpoint,
		Client:  httpClient,
	})
	if err != nil {
		return nil, err
	}

	clientWrapper := createHTTPClient(apiClient)
	return &Client{
		client: clientWrapper,
		tracer: tracer,
		API:    v1.NewAPI(clientWrapper),
	}, nil
}

// wrap the prometheus client with trace context
type httpClient struct {
	api.Client

	propagator propagation.TextMapPropagator
}

func createHTTPClient(c api.Client) *httpClient {
	return &httpClient{
		Client:     c,
		propagator: otel.GetTextMapPropagator(),
	}
}

// Do wraps the api.Client with trace context headers injection
func (ac *httpClient) Do(ctx context.Context, req *http.Request) (*http.Response, []byte, error) {
	ac.propagator.Inject(ctx, propagation.HeaderCarrier(req.Header))
	return ac.Client.Do(ctx, req)
}
