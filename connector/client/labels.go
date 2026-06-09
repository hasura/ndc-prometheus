package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// LabelNames return a list of [label names]
//
// [label names]: https://prometheus.io/docs/prometheus/latest/querying/api/#getting-label-names
func (c *Client) LabelNames(
	ctx context.Context,
	matches []string,
	startTime, endTime time.Time,
	limit uint64,
) ([]string, v1.Warnings, error) {
	endpoint := c.client.URL("/api/v1/labels", nil)
	q := endpoint.Query()

	for _, m := range matches {
		if m == "" {
			continue
		}

		q.Add("match[]", m)
	}

	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}

	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}

	if limit > 0 {
		q.Set("limit", strconv.FormatUint(limit, 10))
	}

	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	body, w, err := c.do(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	var labelNames []string

	err = json.Unmarshal(body, &labelNames)

	return labelNames, w, err
}

// GetLabelValues return a list of [label values] for a given label name.
// This method uses a direct HTTP GET request (like LabelNames) to support
// additional parameters such as match[] filters and limits.
//
// [label values]: https://prometheus.io/docs/prometheus/latest/querying/api/#querying-label-values
func (c *Client) GetLabelValues(
	ctx context.Context,
	label string,
	matches []string,
	startTime, endTime time.Time,
	limit uint64,
) (model.LabelValues, v1.Warnings, error) {
	endpoint := c.client.URL("/api/v1/label/"+label+"/values", nil)
	q := endpoint.Query()

	for _, m := range matches {
		if m == "" {
			continue
		}

		q.Add("match[]", m)
	}

	if !startTime.IsZero() {
		q.Set("start", formatTime(startTime))
	}

	if !endTime.IsZero() {
		q.Set("end", formatTime(endTime))
	}

	if limit > 0 {
		q.Set("limit", strconv.FormatUint(limit, 10))
	}

	endpoint.RawQuery = q.Encode()

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	body, w, err := c.do(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	var labelValues model.LabelValues

	err = json.Unmarshal(body, &labelValues)

	return labelValues, w, err
}
