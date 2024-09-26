package internal

import (
	"context"
	"testing"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"gotest.tools/v3/assert"
)

var testCases = []struct {
	Name        string
	Request     schema.QueryRequest
	Predicate   CollectionRequest
	QueryString string
	IsEmpty     bool
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
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
					},
				},
				"instance": {
					Name: "instance",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					},
				},
			},
			Functions: []KeyValue{
				{Key: "sum", Value: []string{"job"}},
				{Key: "max", Value: []string{}},
				{Key: "abs", Value: true},
			},
		},
		QueryString: `abs(max(sum by (job) (go_gc_duration_seconds{instance=~"localhost:9090|node-exporter:9100",job="node"} offset 5m0s))) >= 0.000000`,
	},
	{
		Name: "label_expressions_empty",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"offset": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					},
				},
			},
		},
		QueryString: "",
		IsEmpty:     true,
	},
	{
		Name: "label_expressions_equal",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"step": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="node"}`,
	},
	{
		Name: "label_expressions_in",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"node|localhost:9090"}`,
	},
	{
		Name: "label_expressions_nin",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_nin", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_nin", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_nin", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_nin", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job!~"localhost:9090|node|prometheus"}`,
	},
	{
		Name: "label_expressions_nin",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), metadata.Regex, schema.NewComparisonValueScalar("node.*")),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), metadata.Regex, schema.NewComparisonValueScalar("node.*")),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"node.*"}`,
	},
	{
		Name: "label_expressions_nin",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp", nil, nil), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), metadata.NotRegex, schema.NewComparisonValueScalar("node.*")),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			Start: schema.NewComparisonValueScalar("2024-09-10T00:00:00Z").Encode(),
			End:   schema.NewComparisonValueScalar("2024-09-11T00:00:00Z").Encode(),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), metadata.NotRegex, schema.NewComparisonValueScalar("node.*")),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job!~"node.*"}`,
	},
	{
		Name: "label_expressions_in_string",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar(`["ndc-prometheus", "node", "prometheus"]`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar(`["ndc-prometheus", "node", "prometheus"]`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"ndc-prometheus|node|prometheus"}`,
	},
	{
		Name: "label_expressions_in_string_pg",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar(`{ndc-prometheus,node,prometheus}`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar(`{ndc-prometheus,node,prometheus}`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"ndc-prometheus|node|prometheus"}`,
	},
	{
		Name: "label_expressions_eq_neq",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					},
				},
			},
		},
		QueryString: ``,
		IsEmpty:     true,
	},
	{
		Name: "label_expressions_eq_in_neq",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{`ndc-prometheus`, "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{`ndc-prometheus`, "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					},
				},
			},
		},
		QueryString: ``,
		IsEmpty:     true,
	},
	{
		Name: "label_expressions_eq_regex",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`.+`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="ndc-prometheus"}`,
	},
	{
		Name: "label_expressions_in_regex",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="ndc-prometheus"}`,
	},
	{
		Name: "label_expressions_in_eq_regex",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"timeout": schema.NewArgumentLiteral("5m").Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job", nil, nil), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
					},
				},
			},
		},
		QueryString: ``,
		IsEmpty:     true,
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

			request, queryString, ok, err := executor.Explain(context.TODO())
			assert.NilError(t, err)
			assert.DeepEqual(t, tc.Predicate, *request)
			assert.Equal(t, tc.QueryString, queryString)
			assert.Equal(t, !tc.IsEmpty, ok)
		})
	}
}
