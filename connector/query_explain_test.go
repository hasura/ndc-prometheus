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

	type ExpectedResponse struct {
		Query  string
		Groups *internal.QueryCollectionGroupingExplainResult
	}

	testCases := []struct {
		RequestFile    string
		ExpectedQuery  string
		ExpectedGroups *internal.QueryCollectionGroupingExplainResult
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
		})
	}
}
