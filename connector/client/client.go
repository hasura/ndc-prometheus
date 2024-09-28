package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"
	"net/http"
	"net/url"
	"strconv"
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
	client api.Client
	tracer trace.Tracer

	v1.API
	clientOptions

	// common OpenTelemetry attributes
	serverAddress string
	serverPort    int
}

// NewClient creates a new Prometheus client instance
func NewClient(ctx context.Context, cfg ClientSettings, tracer trace.Tracer, options ...Option) (*Client, error) {

	endpoint, err := cfg.URL.Get()
	if err != nil {
		return nil, fmt.Errorf("url: %s", err)
	}
	if endpoint == "" {
		return nil, errors.New("the endpoint setting is empty")
	}

	u, err := url.Parse(endpoint)
	if err != nil {
		return nil, fmt.Errorf("invalid Prometheus URL: %s", err)
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

	opts := defaultClientOptions
	for _, opt := range options {
		opt(&opts)
	}
	clientWrapper := createHTTPClient(apiClient)

	c := &Client{
		client:        clientWrapper,
		tracer:        tracer,
		API:           v1.NewAPI(clientWrapper),
		clientOptions: opts,

		serverAddress: u.Host,
	}

	port := u.Port()
	if port != "" {
		p, err := strconv.ParseInt(port, 10, 32)
		if err != nil {
			return nil, fmt.Errorf("invalid port of Prometheus URL: %s", err)
		}
		c.serverPort = int(p)
	}
	return c, nil
}

// ApplyOptions apply options to the Prometheus request
func (c *Client) ApplyOptions(span trace.Span, timeout any) ([]v1.Option, error) {
	timeoutDuration, err := ParseDuration(timeout, c.unixTimeUnit)
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

func (c *Client) do(ctx context.Context, req *http.Request) (*http.Response, []byte, v1.Warnings, error) {
	resp, body, err := c.client.Do(ctx, req)
	if err != nil {
		return resp, body, nil, err
	}

	code := resp.StatusCode

	if code/100 != 2 && !apiError(code) {
		errorType, errorMsg := errorTypeAndMsgFor(resp)
		return resp, body, nil, &v1.Error{
			Type:   errorType,
			Msg:    errorMsg,
			Detail: string(body),
		}
	}

	var result apiResponse

	if http.StatusNoContent != code {
		if jsonErr := json.Unmarshal(body, &result); jsonErr != nil {
			return resp, body, nil, &v1.Error{
				Type: v1.ErrBadResponse,
				Msg:  jsonErr.Error(),
			}
		}
	}

	if apiError(code) && result.Status == "success" {
		err = &v1.Error{
			Type: v1.ErrBadResponse,
			Msg:  "inconsistent body for response code",
		}
	}

	if result.Status == "error" {
		err = &v1.Error{
			Type: result.ErrorType,
			Msg:  result.Error,
		}
	}

	return resp, []byte(result.Data), result.Warnings, err
}

type clientOptions struct {
	timeout      *model.Duration
	unixTimeUnit UnixTimeUnit
}

var defaultClientOptions = clientOptions{
	unixTimeUnit: UnixTimeSecond,
}

// Option the wrapper function to set optional client options
type Option func(opts *clientOptions)

// WithTimeout sets the default timeout option to the client
func WithTimeout(t *model.Duration) Option {
	return func(opts *clientOptions) {
		opts.timeout = t
	}
}

// WithTimeout sets the default timeout option to the client
func WithUnixTimeUnit(u UnixTimeUnit) Option {
	return func(opts *clientOptions) {
		opts.unixTimeUnit = u
	}
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
		slog.Debug(fmt.Sprintf("%s %s", strings.ToUpper(req.Method), req.URL.String()), attrs...)
	}
	return r, bs, err
}
