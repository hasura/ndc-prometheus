package client

import (
	"context"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"strings"
	"time"

	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/client_golang/api"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/propagation"
	"go.opentelemetry.io/otel/trace"
)

// Client extends the Prometheus API client with advanced methods for the Prometheus connector
type Client struct {
	client  api.Client
	tracer  trace.Tracer
	timeout *model.Duration

	v1.API
}

// NewClient creates a new Prometheus client instance
func NewClient(ctx context.Context, cfg ClientSettings, tracer trace.Tracer, timeout *model.Duration) (*Client, error) {

	endpoint, err := cfg.URL.Get()
	if err != nil {
		return nil, fmt.Errorf("url: %s", err)
	}
	if endpoint == "" {
		return nil, errors.New("the endpoint setting is empty")
	}

	httpClient, err := cfg.createHttpClient(ctx)
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
		client:  clientWrapper,
		tracer:  tracer,
		timeout: timeout,
		API:     v1.NewAPI(clientWrapper),
	}, nil
}

// ApplyOptions apply options to the Prometheus request
func (c *Client) ApplyOptions(span trace.Span, timeout any) ([]v1.Option, error) {
	timeoutDuration, err := ParseDuration(timeout)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse timeout")
		span.RecordError(err)
		return nil, fmt.Errorf("failed to decode the timeout parameter: %s", err)
	}

	var options []v1.Option
	if timeoutDuration == 0 && c.timeout != nil {
		timeoutDuration = time.Duration(*c.timeout)
	}

	if timeoutDuration > 0 {
		options = append(options, v1.WithTimeout(timeoutDuration))
	}

	return options, nil
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
	r, bs, err := ac.Client.Do(ctx, req)
	if utils.IsDebug(slog.Default()) {
		attrs := []any{}
		if r != nil {
			attrs = append(attrs, slog.Int("status_code", r.StatusCode))
		}
		if len(bs) > 0 {
			attrs = append(attrs, slog.String("response", string(bs)))
		}
		if err != nil {
			attrs = append(attrs, slog.String("error", err.Error()))
		}
		slog.Debug(fmt.Sprintf("%s %s", strings.ToUpper(req.Method), req.RequestURI), attrs...)
	}
	return r, bs, err
}
