package client

import (
	"context"
	"encoding/json"
	"net/http"
	"strconv"
	"time"

	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// LabelNames return a list of [label names]
//
// [label names]: https://prometheus.io/docs/prometheus/latest/querying/api/#getting-label-names
func (c *Client) LabelNames(ctx context.Context, matches []string, startTime, endTime time.Time, limit uint64) ([]string, v1.Warnings, error) {
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

	req, err := http.NewRequest("GET", endpoint.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	_, body, w, err := c.do(ctx, req)
	if err != nil {
		return nil, nil, err
	}
	var labelNames []string
	err = json.Unmarshal(body, &labelNames)
	return labelNames, w, err

}
