package client

import (
	"context"
	"errors"
	"net/http"
)

// Healthy sends a [health check] request to check
//
// [health check](https://prometheus.io/docs/prometheus/latest/management_api/#health-check)
func (c *Client) Healthy(ctx context.Context) error {
	endpoint := c.client.URL("/-/healthy", map[string]string{})

	req, err := http.NewRequestWithContext(ctx, http.MethodGet, endpoint.String(), nil)
	if err != nil {
		return err
	}

	resp, bs, err := c.client.Do(ctx, req)
	if err != nil {
		return err
	}

	_ = resp.Body.Close()

	if resp.StatusCode == http.StatusOK {
		return nil
	}

	if len(bs) > 0 {
		return errors.New(string(bs))
	}

	return errors.New(resp.Status)
}
