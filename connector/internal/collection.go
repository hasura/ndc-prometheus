package internal

import (
	"context"
	"fmt"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/trace"
)

var valueBinaryOperators = map[string]string{
	metadata.Equal:          "==",
	metadata.NotEqual:       "!=",
	metadata.Least:          "<",
	metadata.LeastOrEqual:   "<=",
	metadata.Greater:        ">",
	metadata.GreaterOrEqual: ">=",
}

type QueryCollectionExecutor struct {
	Client     *client.Client
	Tracer     trace.Tracer
	Runtime    *metadata.RuntimeSettings
	Request    *schema.QueryRequest
	MetricName string
	Metric     metadata.MetricInfo
	Variables  map[string]any
	Arguments  map[string]any
}

// Execute executes the query request.
func (qce *QueryCollectionExecutor) Execute(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
) (*schema.RowSet, error) {
	ctx, span := qce.Tracer.Start(ctx, "Execute Collection")
	defer span.End()

	if !explainResult.OK {
		// early returns zero rows
		// the evaluated query always returns empty values
		return &schema.RowSet{
			Aggregates: schema.RowSetAggregates{},
			Rows:       []map[string]any{},
		}, nil
	}

	nullableFlat, err := utils.DecodeNullableBoolean(qce.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(
			fmt.Sprintf("expected boolean type for the flat field, got: %v", err),
			map[string]any{
				"field": metadata.ArgumentKeyFlat,
			},
		)
	}

	flat := qce.Runtime.IsFlat(nullableFlat)
	results := &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
	}

	if explainResult.QueryString != "" {
		rawResults, err := qce.query(ctx, explainResult.QueryString, explainResult.Request, flat)
		if err != nil {
			return nil, err
		}

		rows, err := utils.EvalObjectsWithColumnSelection(qce.Request.Query.Fields, rawResults)
		if err != nil {
			return nil, err
		}

		results.Rows = rows
	}

	results.Groups, err = qce.queryGroupAggregates(ctx, explainResult)
	if err != nil {
		return nil, err
	}

	return results, nil
}

func (qce *QueryCollectionExecutor) query(
	ctx context.Context,
	queryString string,
	predicate *CollectionRequest,
	flat bool,
) ([]map[string]any, error) {
	if predicate.Range != nil {
		return qce.queryRange(ctx, queryString, predicate, flat)
	}

	return qce.queryInstant(ctx, queryString, predicate, flat)
}

func (qce *QueryCollectionExecutor) queryInstant(
	ctx context.Context,
	queryString string,
	predicate *CollectionRequest,
	flat bool,
) ([]map[string]any, error) {
	vector, _, err := qce.Client.Query(ctx, queryString, predicate.Timestamp, predicate.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	sortVector(vector, predicate.OrderBy)
	vector = paginateVector(vector, qce.Request.Query)
	results := createQueryResultsFromVector(vector, qce.Metric.Labels, qce.Runtime, flat)

	return results, nil
}

func (qce *QueryCollectionExecutor) queryRange(
	ctx context.Context,
	queryString string,
	predicate *CollectionRequest,
	flat bool,
) ([]map[string]any, error) {
	matrix, err := qce.queryRangeMatrix(ctx, queryString, predicate)
	if err != nil {
		return nil, err
	}

	sortMatrix(matrix, predicate.OrderBy)
	results := createQueryResultsFromMatrix(matrix, qce.Metric.Labels, qce.Runtime, flat)

	return paginateQueryResults(results, qce.Request.Query), nil
}

func (qce *QueryCollectionExecutor) queryRangeMatrix(
	ctx context.Context,
	queryString string,
	predicate *CollectionRequest,
) (model.Matrix, error) {
	matrix, _, err := qce.Client.QueryRange(ctx, queryString, *predicate.Range, predicate.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	return matrix, nil
}
