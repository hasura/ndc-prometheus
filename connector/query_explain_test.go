package connector

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/hasura/ndc-sdk-go/connector"
	"github.com/hasura/ndc-sdk-go/schema"
	"gotest.tools/v3/assert"
)

func TestQueryExplain(t *testing.T) {
	setDefaultEnvironments(t)

	server, err := connector.NewServer(NewPrometheusConnector(), &connector.ServerOptions{
		Configuration: "../tests/configuration",
	})
	assert.NilError(t, err)

	httpServer := server.BuildTestServer()
	defer httpServer.Close()

	testCases := []struct {
		RequestFile     string
		ExpectedQueries map[string]string
	}{
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_sum_rate/request.json",
			ExpectedQueries: map[string]string{
				"group_aggregates[sum(app.process_cpu_seconds_total_rate.value)]": "sum by (timestamp, job, instance) (rate(process_cpu_seconds_total[15s]))",
			},
		},
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_aggregate_count/request.json",
			ExpectedQueries: map[string]string{
				"aggregates[value__count]":          "count(process_cpu_seconds_total)",
				"aggregates[value__count_distinct]": "count(process_cpu_seconds_total)",
				"aggregates[value_avg]":             "avg(process_cpu_seconds_total)",
				"aggregates[value_max]":             "max(process_cpu_seconds_total)",
				"aggregates[value_min]":             "min(process_cpu_seconds_total)",
				"aggregates[value_sum]":             "sum(process_cpu_seconds_total)",
			},
		},
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_rate_aggregate/request.json",
			ExpectedQueries: map[string]string{
				"aggregates[value__count]":          "count(rate(process_cpu_seconds_total[1m]))",
				"aggregates[value__count_distinct]": "count(rate(process_cpu_seconds_total[1m]))",
				"aggregates[value_avg]":             "avg(rate(process_cpu_seconds_total[1m]))",
				"aggregates[value_max]":             "max(rate(process_cpu_seconds_total[1m]))",
				"aggregates[value_min]":             "min(rate(process_cpu_seconds_total[1m]))",
				"aggregates[value_sum]":             "sum(rate(process_cpu_seconds_total[1m]))",
			},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.RequestFile, func(t *testing.T) {
			rawBytes, err := os.ReadFile(tc.RequestFile)
			assert.NilError(t, err)

			res, err := http.DefaultClient.Post(httpServer.URL+"/query/explain", "application/json", bytes.NewReader(rawBytes))
			assert.NilError(t, err)
			defer res.Body.Close()

			var result schema.ExplainResponse
			assert.NilError(t, json.NewDecoder(res.Body).Decode(&result))
			assert.DeepEqual(t, tc.ExpectedQueries, map[string]string(result.Details))
		})
	}
}
