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

// Series returns the list of [time series] that match a certain label set.
// Google Managed Prometheus supports GET method only so the base API library doesn't work.
// [time series](https://prometheus.io/docs/prometheus/latest/querying/api/#finding-series-by-label-matchers)
func (c *Client) Series(ctx context.Context, matches []string, startTime, endTime time.Time, limit uint64) ([]model.LabelSet, v1.Warnings, error) {
	u := c.client.URL("/api/v1/series", nil)
	q := u.Query()

	for _, m := range matches {
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
	u.RawQuery = q.Encode()

	req, err := http.NewRequest("GET", u.String(), nil)
	if err != nil {
		return nil, nil, err
	}

	_, body, warnings, err := c.do(ctx, req)
	if err != nil {
		return nil, nil, err
	}

	var mset []model.LabelSet
	return mset, warnings, json.Unmarshal(body, &mset)
}

func formatTime(t time.Time) string {
	return strconv.FormatFloat(float64(t.Unix())+float64(t.Nanosecond())/1e9, 'f', -1, 64)
}
