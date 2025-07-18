package internal

import (
	"testing"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"gotest.tools/v3/assert"
)

var testCases = []struct {
	Name        string
	MetricName  string
	Request     schema.QueryRequest
	Predicate   CollectionRequest
	QueryString string
	ErrorMsg    string
	IsEmpty     bool
	Groups      *QueryCollectionGroupingExplainResult
	Aggregates  map[string]string
	Functions   []KeyValue
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
					{
						"sort_by_label_desc": []string{"job"},
					},
					{
						"limitk": 2,
					},
				}).Encode(),
			},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
					),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value"), "_gte", schema.NewComparisonValueScalar("0")),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			Value: schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value"), "_gte", schema.NewComparisonValueScalar("0")),
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
					},
				},
				"instance": {
					Name: "instance",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					},
				},
			},
			Functions: []KeyValue{
				{Key: "sum", Value: []string{"job"}},
				{Key: "max", Value: []string{}},
				{Key: "abs", Value: true},
				{Key: "sort_by_label_desc", Value: []string{"job"}},
				{Key: "limitk", Value: 2},
			},
		},
		QueryString: `limitk(2, sort_by_label_desc(abs(max(sum by (job) (go_gc_duration_seconds{instance=~"localhost:9090|node-exporter:9100",job="node"} offset 5m0s))), "job")) >= 0.000000`,
		Aggregates:  map[string]string{},
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
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
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
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="node"}`,
		Aggregates:  map[string]string{},
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
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"node|localhost:9090"}`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "label_expressions_nin",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_nin", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_nin", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_nin", schema.NewComparisonValueScalar([]string{"node", "localhost:9090", "prometheus"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_nin", schema.NewComparisonValueScalar([]string{"localhost:9090", "node"})),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job!~"localhost:9090|node|prometheus"}`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "label_expressions_regex",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), metadata.Regex, schema.NewComparisonValueScalar("node.*")),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), metadata.Regex, schema.NewComparisonValueScalar("node.*")),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"node.*"}`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "label_expressions_nregex",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments:  schema.QueryRequestArguments{},
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionAnd(
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), metadata.NotRegex, schema.NewComparisonValueScalar("node.*")),
					),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2024, 9, 10, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2024, 9, 11, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), metadata.NotRegex, schema.NewComparisonValueScalar("node.*")),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job!~"node.*"}`,
		Aggregates:  map[string]string{},
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar(`["ndc-prometheus", "node", "prometheus"]`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar(`["ndc-prometheus", "node", "prometheus"]`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job=~"ndc-prometheus|node|prometheus"}`,
		Aggregates:  map[string]string{},
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{`ndc-prometheus`, "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{`ndc-prometheus`, "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_neq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar(`ndc-prometheus`)),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`.+`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="ndc-prometheus"}`,
		Aggregates:  map[string]string{},
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
					},
				},
			},
		},
		QueryString: `go_gc_duration_seconds{job="ndc-prometheus"}`,
		Aggregates:  map[string]string{},
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
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
					schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
				).Encode(),
			},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Timeout: 5 * time.Minute,
			},
			LabelExpressions: map[string]*LabelExpression{
				"job": {
					Name: "job",
					Expressions: []schema.ExpressionBinaryComparisonOperator{
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_in", schema.NewComparisonValueScalar([]string{"ndc-prometheus", "node"})),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						*schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_regex", schema.NewComparisonValueScalar(`ndc-.+`)),
					},
				},
			},
		},
		QueryString: ``,
		IsEmpty:     true,
	},
	{
		Name: "aggregation_histogram_fraction",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"offset": schema.NewArgumentLiteral("5m").Encode(),
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"quantile": 0.1,
					},
					{
						"round": 0.2,
					},
					{
						"clamp": map[string]any{
							"min": 1,
							"max": 2,
						},
					},
					{
						"histogram_fraction": map[string]any{
							"min": 0.1,
							"max": 0.2,
						},
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "quantile", Value: 0.1},
				{Key: "round", Value: 0.2},
				{Key: "clamp", Value: map[string]any{
					"min": 1,
					"max": 2,
				}},
				{Key: "histogram_fraction", Value: map[string]any{
					"min": 0.1,
					"max": 0.2,
				}},
			},
		},
		QueryString: `histogram_fraction(0.100000, 0.200000, clamp(round(quantile(0.100000, go_gc_duration_seconds offset 5m0s), 0.200000), 1.000000, 2.000000))`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_holt_winters",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"holt_winters": map[string]any{
							"sf":    0.1,
							"tf":    0.2,
							"range": "1m",
						},
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "holt_winters", Value: map[string]any{
					"sf":    0.1,
					"tf":    0.2,
					"range": "1m",
				}},
			},
		},
		QueryString: `holt_winters(go_gc_duration_seconds[1m], 0.100000, 0.200000)`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_predict_linear",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"predict_linear": map[string]any{
							"t":     0.1,
							"range": 60,
						},
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "predict_linear", Value: map[string]any{
					"t":     0.1,
					"range": 60,
				}},
			},
		},
		QueryString: `predict_linear(go_gc_duration_seconds[1m], 0.100000)`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_quantile_over_time",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"quantile_over_time": map[string]any{
							"quantile": 0.1,
							"range":    "1m",
						},
					},
					{
						"label_join": map[string]any{
							"dest_label":    "dest",
							"source_labels": []string{"job", "instance"},
							"separator":     "-",
						},
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "quantile_over_time", Value: map[string]any{
					"quantile": 0.1,
					"range":    "1m",
				}},
				{Key: "label_join", Value: map[string]any{
					"dest_label":    "dest",
					"source_labels": []string{"job", "instance"},
					"separator":     "-",
				}},
			},
		},
		QueryString: `label_join(quantile_over_time(0.100000, go_gc_duration_seconds[1m]), "dest", "-", "job", "instance")`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_label_replace",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"label_replace": map[string]any{
							"dest_label":   "dest",
							"source_label": "job",
							"replacement":  "",
							"regex":        ".+",
						},
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "label_replace", Value: map[string]any{
					"dest_label":   "dest",
					"source_label": "job",
					"replacement":  "",
					"regex":        ".+",
				}},
			},
		},
		QueryString: `label_replace(go_gc_duration_seconds, "dest", "", "job", ".+")`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_clamp_max",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"clamp_max": 1,
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "clamp_max", Value: 1},
			},
		},
		QueryString: `clamp_max(go_gc_duration_seconds, 1.000000)`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_count_values",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"count_values": "job",
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "count_values", Value: "job"},
			},
		},
		QueryString: `count_values("job", go_gc_duration_seconds)`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "aggregation_irate",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"irate": "1m",
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "irate", Value: "1m"},
			},
		},
		QueryString: `irate(go_gc_duration_seconds[1m])`,
		Aggregates:  map[string]string{},
	},
	{
		Name: "invalid_function",
		Request: schema.QueryRequest{
			Collection: "go_gc_duration_seconds",
			Arguments: schema.QueryRequestArguments{
				"fn": schema.NewArgumentLiteral([]map[string]any{
					{
						"test": "1m",
					},
				}).Encode(),
			},
			Query: schema.Query{},
		},
		Predicate: CollectionRequest{
			LabelExpressions: map[string]*LabelExpression{},
			Functions: []KeyValue{
				{Key: "test", Value: "1m"},
			},
		},
		QueryString: ``,
		ErrorMsg:    "failed to evaluate the query",
	},
	{
		Name: "group_star_count",
		Request: schema.QueryRequest{
			Collection: "ndc_prometheus_query_total",
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_gte",
						schema.NewComparisonValueScalar("2025-06-19"),
					),
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_lte",
						schema.NewComparisonValueScalar("2025-06-25"),
					),
				).Encode(),
				Groups: &schema.Grouping{
					Aggregates: schema.GroupingAggregates{
						"count": schema.NewAggregateStarCount().Encode(),
					},
					Dimensions: []schema.Dimension{
						schema.NewDimensionColumn("job", nil).Encode(),
						schema.NewDimensionColumn("instance", nil).Encode(),
					},
				},
			},
			Arguments:               schema.QueryRequestArguments{},
			CollectionRelationships: schema.QueryRequestCollectionRelationships{},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2025, 06, 25, 0, 0, 0, 0, time.UTC),
					Step:  30 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{},
			Groups: &Grouping{
				Dimensions: []string{"job", "instance"},
				Aggregates: schema.GroupingAggregates{
					"count": schema.NewAggregateStarCount().Encode(),
				},
			},
		},
		Groups: &QueryCollectionGroupingExplainResult{
			Dimensions: []string{"job", "instance"},
			AggregateQueries: map[string]string{
				"count": "count by (job, instance) (ndc_prometheus_query_total)",
			},
		},
		Aggregates: map[string]string{},
	},
	{
		Name:       "counter_increase",
		MetricName: "ndc_prometheus_query_total",
		Functions: []KeyValue{
			{
				Key:   string(metadata.Increase),
				Value: "5m",
			},
		},
		Request: schema.QueryRequest{
			Collection: "ndc_prometheus_query_total_increase",
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_gte",
						schema.NewComparisonValueScalar("2025-06-19"),
					),
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_lte",
						schema.NewComparisonValueScalar("2025-06-25"),
					),
				).Encode(),
			},
			Arguments: schema.QueryRequestArguments{
				"step": schema.NewArgumentLiteral("5m").Encode(),
			},
			CollectionRelationships: schema.QueryRequestCollectionRelationships{},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2025, 06, 25, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{},
			Groups:           &Grouping{},
		},
		QueryString: "increase(ndc_prometheus_query_total[5m])",
		Aggregates:  map[string]string{},
	},
	{
		Name:       "counter_sum_increase",
		MetricName: "ndc_prometheus_query_total",
		Functions: []KeyValue{
			{
				Key:   string(metadata.Increase),
				Value: "5m",
			},
		},
		Request: schema.QueryRequest{
			Collection: "ndc_prometheus_query_total_increase",
			Query: schema.Query{
				Predicate: schema.NewExpressionAnd(
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_gte",
						schema.NewComparisonValueScalar("2025-06-19"),
					),
					schema.NewExpressionBinaryComparisonOperator(
						schema.NewComparisonTargetColumn("timestamp"),
						"_lte",
						schema.NewComparisonValueScalar("2025-06-25"),
					),
				).Encode(),
				Groups: &schema.Grouping{
					Aggregates: schema.GroupingAggregates{
						"sum": schema.NewAggregateSingleColumn("value", "sum").Encode(),
					},
					Dimensions: []schema.Dimension{
						schema.NewDimensionColumn("job", nil).Encode(),
						schema.NewDimensionColumn("instance", nil).Encode(),
					},
				},
			},
			Arguments: schema.QueryRequestArguments{
				"step": schema.NewArgumentLiteral("5m").Encode(),
			},
			CollectionRelationships: schema.QueryRequestCollectionRelationships{},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{
				Range: &v1.Range{
					Start: time.Date(2025, 06, 19, 0, 0, 0, 0, time.UTC),
					End:   time.Date(2025, 06, 25, 0, 0, 0, 0, time.UTC),
					Step:  5 * time.Minute,
				},
			},
			LabelExpressions: map[string]*LabelExpression{},
			Groups: &Grouping{
				Dimensions: []string{"job", "instance"},
				Aggregates: schema.GroupingAggregates{
					"sum": schema.NewAggregateSingleColumn("value", "sum").Encode(),
				},
			},
		},
		Groups: &QueryCollectionGroupingExplainResult{
			Dimensions: []string{"job", "instance"},
			AggregateQueries: map[string]string{
				"sum": "sum by (job, instance) (increase(ndc_prometheus_query_total[5m]))",
			},
		},
		Aggregates: map[string]string{},
	},
	{
		Name:       "aggregate_count",
		MetricName: "process_cpu_seconds_total",
		Request: schema.QueryRequest{
			Collection: "process_cpu_seconds_total",
			Query: schema.Query{
				Aggregates: schema.QueryAggregates{
					"value__count": schema.NewAggregateColumnCount("value", false).Encode(),
				},
			},
			Arguments:               schema.QueryRequestArguments{},
			CollectionRelationships: schema.QueryRequestCollectionRelationships{},
		},
		Predicate: CollectionRequest{
			CollectionValidatedArguments: CollectionValidatedArguments{},
			LabelExpressions:             map[string]*LabelExpression{},
		},
		Aggregates: map[string]string{
			"value__count": "count(process_cpu_seconds_total)",
		},
	},
}

func TestCollectionQueryExplain(t *testing.T) {
	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			arguments, err := utils.ResolveArgumentVariables(tc.Request.Arguments, map[string]any{})
			assert.NilError(t, err)

			metricName := tc.MetricName
			if metricName == "" {
				metricName = tc.Request.Collection
			}

			executor := &QueryCollectionExecutor{
				Request:    &tc.Request,
				MetricName: metricName,
				Variables:  map[string]any{},
				Arguments:  arguments,
				Runtime:    &metadata.RuntimeSettings{},
			}

			validatedRequest, err := EvalCollectionRequest(&tc.Request, arguments, executor.Variables, executor.Runtime)
			assert.NilError(t, err)

			validatedRequest.Functions = append(tc.Functions, validatedRequest.Functions...)

			result, err := executor.Explain(validatedRequest)
			if tc.ErrorMsg != "" {
				assert.ErrorContains(t, err, tc.ErrorMsg)

				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, tc.Predicate.Value, result.Request.Value)
			assert.DeepEqual(t, tc.Predicate.Range, result.Request.Range)
			assert.DeepEqual(t, tc.Predicate.Timestamp, result.Request.Timestamp)
			assert.DeepEqual(t, tc.Predicate.Timeout, result.Request.Timeout)
			assert.Equal(t, tc.QueryString, result.QueryString)
			assert.Equal(t, !tc.IsEmpty, result.OK)
			assert.DeepEqual(t, tc.Aggregates, result.Aggregates)

			if tc.Groups != nil {
				assert.DeepEqual(t, tc.Groups, result.Groups)
			}
		})
	}
}

func TestCollectionQueryExplainHistogramQuantile(t *testing.T) {
	var hqTestCases = []struct {
		Name        string
		MetricName  string
		Request     schema.QueryRequest
		QueryString string
		ErrorMsg    string
		IsEmpty     bool
		Groups      *QueryCollectionGroupingExplainResult
	}{
		{
			Name: "histogram_quantile",
			Request: schema.QueryRequest{
				Collection: "hasura_graphql_execution_time_seconds_bucket",
				Arguments: schema.QueryRequestArguments{
					"offset":   schema.NewArgumentLiteral("5m").Encode(),
					"quantile": schema.NewArgumentLiteral(0.9).Encode(),
				},
				Query: schema.Query{
					Predicate: schema.NewExpressionAnd(
						schema.NewExpressionAnd(
							schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
							schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value"), "_gte", schema.NewComparisonValueScalar("0")),
					).Encode(),
				},
			},
			QueryString: `histogram_quantile(0.900000, rate(hasura_graphql_execution_time_seconds_bucket{instance=~"localhost:9090|node-exporter:9100",job="node"}[5m] offset 5m0s) >= 0.000000)`,
		},
		{
			Name: "histogram_quantile_sum",
			Request: schema.QueryRequest{
				Collection: "hasura_graphql_execution_time_seconds_bucket",
				Arguments: schema.QueryRequestArguments{
					"offset":   schema.NewArgumentLiteral("5m").Encode(),
					"quantile": schema.NewArgumentLiteral(0.95).Encode(),
				},
				Query: schema.Query{
					Predicate: schema.NewExpressionAnd(
						schema.NewExpressionAnd(
							schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_lt", schema.NewComparisonValueScalar("2024-09-11T00:00:00Z")),
							schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("timestamp"), "_gt", schema.NewComparisonValueScalar("2024-09-10T00:00:00Z")),
						),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("job"), "_eq", schema.NewComparisonValueScalar("node")),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("instance"), "_in", schema.NewComparisonValueScalar([]string{"localhost:9090", "node-exporter:9100"})),
						schema.NewExpressionBinaryComparisonOperator(*schema.NewComparisonTargetColumn("value"), "_gte", schema.NewComparisonValueScalar("0")),
					).Encode(),
					Groups: &schema.Grouping{
						Aggregates: schema.GroupingAggregates{
							"sum": schema.NewAggregateSingleColumn("value", "sum").Encode(),
						},
						Dimensions: []schema.Dimension{
							schema.NewDimensionColumn("job", nil).Encode(),
							schema.NewDimensionColumn("instance", nil).Encode(),
						},
					},
				},
			},
			Groups: &QueryCollectionGroupingExplainResult{
				Dimensions: []string{"job", "instance", "le"},
				AggregateQueries: map[string]string{
					"sum": `histogram_quantile(0.950000, sum by (job, instance, le) (rate(hasura_graphql_execution_time_seconds_bucket{instance=~"localhost:9090|node-exporter:9100",job="node"}[5m] offset 5m0s) >= 0.000000))`,
				},
			},
		},
	}

	for _, tc := range hqTestCases {
		t.Run(tc.Name, func(t *testing.T) {
			arguments, err := utils.ResolveArgumentVariables(tc.Request.Arguments, map[string]any{})
			assert.NilError(t, err)

			metricName := tc.MetricName
			if metricName == "" {
				metricName = tc.Request.Collection
			}

			executor := &QueryCollectionExecutor{
				Request:    &tc.Request,
				MetricName: metricName,
				Variables:  map[string]any{},
				Arguments:  arguments,
				Runtime:    &metadata.RuntimeSettings{},
			}

			validatedRequest, err := EvalCollectionRequest(&tc.Request, arguments, executor.Variables, executor.Runtime)
			assert.NilError(t, err)

			result, err := executor.ExplainHistogramQuantile(validatedRequest)
			if tc.ErrorMsg != "" {
				assert.ErrorContains(t, err, tc.ErrorMsg)

				return
			}

			assert.NilError(t, err)
			assert.Equal(t, tc.QueryString, result.QueryString)
			assert.Equal(t, !tc.IsEmpty, result.OK)

			if tc.Groups != nil {
				assert.DeepEqual(t, tc.Groups, result.Groups)
			}
		})
	}
}
