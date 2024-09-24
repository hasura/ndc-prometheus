package internal

import (
	"context"
	"testing"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"gotest.tools/v3/assert"
)

var testCases = []struct {
	Name        string
	Request     schema.QueryRequest
	Predicate   CollectionRequest
	QueryString string
}{
	{
		Name: "nested_expressions",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"offset": schema.NewArgumentLiteral("5m").Encode(),
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"sum": []string{"job"},
					},
					{
						"max": []string{},
					},
					{
						"abs": true,
					},
				}).Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
					),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value", nil, nil), "_gte", schema.NewComparisonValueScalar("0")),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			Value: schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value", nil, nil), "_gte", schema.NewComparisonValueScalar("0")),
			LabelExpressions: map[string]*schema.ExpressionBinaryComparisonOperator{
				"job":      schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
				"instance": schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
			},
			Functions: []KeyValue{
				{Key: "sum", Value: []string{"job"}},
				{Key: "max", Value: []string{}},
				{Key: "abs", Value: true},
			},
		},
		QueryString: `abs(max(sum by (job) (go_gc_duration_seconds{instance=~"localhost:9090|node-exporter:9100",job="node"} offset 5m0s))) >= 0.000000`,
	},
}

func TestCollectionQueryExplain(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			arguments, err := utils.ResolveArgumentVariables(tc.Request.Arguments, map[string]any{})
			assert.NilError(t, err)

			executor := &QueryCollectionExecutor{
				Request:   &tc.Request,
				Variables: map[string]any{},
				Arguments: arguments,
			}

			request, queryString, err := executor.Explain(context.TODO())
			assert.NilError(t, err)
			assert.DeepEqual(t, tc.Predicate, *request)
			assert.Equal(t, tc.QueryString, queryString)
		})
	}
}
