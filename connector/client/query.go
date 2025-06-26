package client

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"net/http"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// Query evaluates an [instant query] at a single point in time
//
// [instant query]: https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func (c *Client) Query(
	ctx context.Context,
	queryString string,
	ts *time.Time,
	timeout time.Duration,
) (model.Vector, v1.Warnings, error) {
	ctx, span := clientTracer.Start(ctx, "Query")
	defer span.End()

	if ts == nil {
		t := time.Now()
		ts = &t
	}

	c.setQuerySpanAttributes(span, queryString)
	span.SetAttributes(
		attribute.String("timestamp", ts.String()),
		attribute.String("timeout", timeout.String()),
	)

	opts, err := c.ApplyOptions(span, timeout)
	if err != nil {
		return nil, nil, err
	}

	if ts == nil {
		now := time.Now()
		ts = &now
	}

	r, warnings, err := c.API.Query(ctx, queryString, *ts, opts...)

	if len(warnings) > 0 {
		span.SetAttributes(attribute.StringSlice("warnings", warnings))
	}

	if err != nil {
		span.SetStatus(codes.Error, "error querying prometheus")
		span.RecordError(err)

		return nil, warnings, fmt.Errorf("error querying prometheus: %w", err)
	}

	if result, ok := r.(model.Vector); ok {
		return result, warnings, nil
	}

	err = errors.New("did not receive an instant vector result")
	span.SetStatus(codes.Error, err.Error())

	return nil, warnings, err
}

// QueryRange evaluates a [range query] that performs query over a range of time
//
// [range query](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries)
func (c *Client) QueryRange(
	ctx context.Context,
	queryString string,
	timeRange v1.Range,
	timeout time.Duration,
) (model.Matrix, v1.Warnings, error) {
	ctx, span := clientTracer.Start(ctx, "QueryRange")
	defer span.End()

	c.setQuerySpanAttributes(span, queryString)
	span.SetAttributes(attribute.String("start", timeRange.Start.String()))
	span.SetAttributes(attribute.String("end", timeRange.End.String()))
	span.SetAttributes(attribute.String("step", timeRange.Step.String()))
	span.SetAttributes(attribute.String("timeout", timeout.String()))

	opts, err := c.ApplyOptions(span, timeout)
	if err != nil {
		return nil, nil, err
	}

	// execute query
	result, warnings, err := c.API.QueryRange(ctx, queryString, timeRange, opts...)
	if len(warnings) > 0 {
		span.SetAttributes(attribute.StringSlice("warnings", warnings))
	}

	if err != nil {
		span.SetStatus(codes.Error, "failed to run QueryRange")
		span.RecordError(err)

		return nil, warnings, err
	}

	if result, ok := result.(model.Matrix); ok {
		return result, warnings, err
	} else {
		err := errors.New("did not receive a range result")
		span.SetStatus(codes.Error, err.Error())

		return nil, warnings, err
	}
}

type formatQueryResult struct {
	Status string `json:"status"`
	Data   string `json:"data"`
}

// FormatQuery [formats a PromQL expression] in a prettified way
//
// [formats a PromQL expression](https://prometheus.io/docs/prometheus/latest/querying/api/#formatting-query-expressions)
func (c *Client) FormatQuery(ctx context.Context, queryString string) (string, error) {
	endpoint := c.client.URL("/api/v1/format_query", map[string]string{})

	q := endpoint.Query()
	q.Set("query", queryString)
	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return "", err
	}

	resp, bodyResp, err := c.client.Do(ctx, req)
	if err != nil {
		return "", err
	}

	_ = resp.Body.Close()

	if resp.StatusCode >= 400 {
		if len(bodyResp) > 0 {
			return "", errors.New(string(bodyResp))
		}

		return "", errors.New(resp.Status)
	}

	var result formatQueryResult

	if err := json.Unmarshal(bodyResp, &result); err != nil {
		return "", err
	}

	if result.Status != "success" {
		return "", fmt.Errorf("received failed response: %s", string(bodyResp))
	}

	return result.Data, nil
}

func (c *Client) setQuerySpanAttributes(span trace.Span, queryString string) {
	span.SetAttributes(
		attribute.String("db.system", "prometheus"),
		attribute.String("db.query.text", queryString),
		attribute.String("server.address", c.serverAddress),
	)

	if c.serverPort > 0 {
		span.SetAttributes(attribute.Int("server.port", c.serverPort))
	}
}
