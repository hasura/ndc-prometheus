package connector

import (
	"bytes"
	"encoding/json"
	"net/http"
	"os"
	"testing"

	"github.com/hasura/ndc-prometheus/connector/internal"
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
		RequestFile        string
		ExpectedQuery      string
		ExpectedAggregates map[string]string
		ExpectedGroups     *internal.QueryCollectionGroupingExplainResult
	}{
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_sum_rate/request.json",
			ExpectedGroups: &internal.QueryCollectionGroupingExplainResult{
				Dimensions: []string{"timestamp", "job", "instance"},
				AggregateQueries: map[string]string{
					"sum(app.process_cpu_seconds_total_rate.value)": "sum by (timestamp, job, instance) (rate(process_cpu_seconds_total[15s]))",
				},
			},
		},
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_aggregate_count/request.json",
			ExpectedAggregates: map[string]string{
				"value__count":          "count(process_cpu_seconds_total)",
				"value__count_distinct": "count(process_cpu_seconds_total)",
				"value_avg":             "avg(process_cpu_seconds_total)",
				"value_max":             "max(process_cpu_seconds_total)",
				"value_min":             "min(process_cpu_seconds_total)",
				"value_sum":             "sum(process_cpu_seconds_total)",
			},
		},
		{
			RequestFile: "testdata/query/process_cpu_seconds_total_rate_aggregate/request.json",
			ExpectedAggregates: map[string]string{
				"value__count":          "count(rate(process_cpu_seconds_total[1m]))",
				"value__count_distinct": "count(rate(process_cpu_seconds_total[1m]))",
				"value_avg":             "avg(rate(process_cpu_seconds_total[1m]))",
				"value_max":             "max(rate(process_cpu_seconds_total[1m]))",
				"value_min":             "min(rate(process_cpu_seconds_total[1m]))",
				"value_sum":             "sum(rate(process_cpu_seconds_total[1m]))",
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
			assert.Equal(t, tc.ExpectedQuery, result.Details["query"])

			if strGroups, ok := result.Details["groups"]; ok {
				var groups internal.QueryCollectionGroupingExplainResult
				assert.NilError(t, json.Unmarshal([]byte(strGroups), &groups))
				assert.DeepEqual(t, *tc.ExpectedGroups, groups)
			}

			if strAggregates, ok := result.Details["aggregates"]; ok {
				var aggregates map[string]string
				assert.NilError(t, json.Unmarshal([]byte(strAggregates), &aggregates))
				assert.DeepEqual(t, tc.ExpectedAggregates, aggregates)
			}
		})
	}
}
