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
)

// Query evaluates an [instant query] at a single point in time
//
// [instant query]: https://prometheus.io/docs/prometheus/latest/querying/api/#instant-queries
func (c *Client) Query(ctx context.Context, queryString string, timestamp any, timeout any) (model.Vector, v1.Warnings, error) {

	ctx, span := c.tracer.Start(ctx, "Query")
	defer span.End()

	span.SetAttributes(attribute.String("query", queryString))
	span.SetAttributes(attribute.String("timestamp", fmt.Sprint(timestamp)))
	span.SetAttributes(attribute.String("timeout", fmt.Sprint(timeout)))

	opts, err := c.ApplyOptions(span, timeout)
	if err != nil {
		return nil, nil, err
	}

	ts, err := ParseTimestamp(timestamp)
	if err != nil {
		span.SetStatus(codes.Error, "failed to decode time")
		span.RecordError(err)
		return nil, nil, fmt.Errorf("failed to decode time: %s", err)
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
		return nil, warnings, fmt.Errorf("error querying prometheus: %v", err)
	}

	if result, ok := r.(model.Vector); ok {
		return result, warnings, nil
	}

	err = fmt.Errorf("did not receive an instant vector result")
	span.SetStatus(codes.Error, err.Error())
	return nil, warnings, err
}

// QueryRange evaluates a [range query] that performs query over a range of time
//
// [range query](https://prometheus.io/docs/prometheus/latest/querying/api/#range-queries)
func (c *Client) QueryRange(ctx context.Context, queryString string, start any, end any, step any, timeout any) (model.Matrix, v1.Warnings, error) {
	ctx, span := c.tracer.Start(ctx, "QueryRange")
	defer span.End()

	span.SetAttributes(attribute.String("query", queryString))
	span.SetAttributes(attribute.String("start", fmt.Sprint(start)))
	span.SetAttributes(attribute.String("end", fmt.Sprint(end)))
	span.SetAttributes(attribute.String("step", fmt.Sprint(step)))
	span.SetAttributes(attribute.String("timeout", fmt.Sprint(timeout)))

	opts, err := c.ApplyOptions(span, timeout)
	if err != nil {
		return nil, nil, err
	}

	r, err := c.getRange(start, end, step)
	if err != nil {
		span.SetStatus(codes.Error, "failed to parse range")
		span.RecordError(err)
		return nil, nil, err
	}

	span.SetAttributes(attribute.String("start", r.Start.String()))
	span.SetAttributes(attribute.String("end", r.End.String()))
	span.SetAttributes(attribute.String("step", r.Step.String()))

	// execute query
	result, warnings, err := c.API.QueryRange(ctx, queryString, *r, opts...)
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
		err := fmt.Errorf("did not receive a range result")
		span.SetStatus(codes.Error, err.Error())
		return nil, warnings, err
	}
}

func (c *Client) getRange(start any, end any, step any) (*v1.Range, error) {
	result := v1.Range{
		End: time.Now(),
	}
	startTime, err := ParseTimestamp(start)
	if err != nil {
		return nil, err
	}
	if startTime != nil {
		result.Start = *startTime
	}

	// If the user provided an end value, parse it to a time struct and override the default
	endTime, err := ParseTimestamp(end)
	if err != nil {
		return nil, err
	}

	if endTime != nil {
		result.End = *endTime
	}

	// Set up defaults for the step value
	result.Step, err = ParseDuration(step)
	if err != nil {
		return nil, fmt.Errorf("unable to parse step: %s", err)
	}

	if result.Step == 0 {
		result.Step = evalStepFromRange(result.Start, result.End)
	}

	return &result, nil
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

	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return "", err
	}

	resp, bodyResp, err := c.client.Do(ctx, req)
	if err != nil {
		return "", err
	}
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
