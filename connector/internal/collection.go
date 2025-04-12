package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"go.opentelemetry.io/otel/trace"
)

type QueryCollectionExecutor struct {
	Client    *client.Client
	Tracer    trace.Tracer
	Runtime   *metadata.RuntimeSettings
	Request   *schema.QueryRequest
	Metric    metadata.MetricInfo
	Variables map[string]any
	Arguments map[string]any
}

// Explain explains the query request.
func (qce *QueryCollectionExecutor) Explain(
	ctx context.Context,
) (*CollectionRequest, string, bool, error) {
	expressions, err := EvalCollectionRequest(qce.Request, qce.Arguments)
	if err != nil {
		return nil, "", false, schema.UnprocessableContentError(err.Error(), map[string]any{
			"collection": qce.Request.Collection,
		})
	}

	queryString, ok, err := qce.buildQueryString(expressions)
	if err != nil {
		return nil, "", false, schema.UnprocessableContentError(
			"failed to evaluate the query",
			map[string]any{
				"collection": qce.Request.Collection,
				"error":      err.Error(),
			},
		)
	}

	return expressions, queryString, ok, nil
}

// Execute executes the query request.
func (qce *QueryCollectionExecutor) Execute(ctx context.Context) (*schema.RowSet, error) {
	ctx, span := qce.Tracer.Start(ctx, "Execute Collection")
	defer span.End()

	expressions, queryString, ok, err := qce.Explain(ctx)
	if err != nil {
		return nil, err
	}

	if !ok {
		// early returns zero rows
		// the evaluated query always returns empty values
		return &schema.RowSet{
			Aggregates: schema.RowSetAggregates{},
			Rows:       []map[string]any{},
		}, nil
	}

	flat, err := utils.DecodeNullableBoolean(qce.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(
			fmt.Sprintf("expected boolean type for the flat field, got: %v", err),
			map[string]any{
				"field": metadata.ArgumentKeyFlat,
			},
		)
	}

	if flat == nil {
		flat = &qce.Runtime.Flat
	}

	var rawResults []map[string]any

	if expressions.Timestamp != nil {
		rawResults, err = qce.queryInstant(ctx, queryString, expressions, *flat)
	} else {
		rawResults, err = qce.queryRange(ctx, queryString, expressions, *flat)
	}

	if err != nil {
		return nil, err
	}

	result, err := utils.EvalObjectsWithColumnSelection(qce.Request.Query.Fields, rawResults)
	if err != nil {
		return nil, err
	}

	return &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
		Rows:       result,
	}, nil
}

func (qce *QueryCollectionExecutor) queryInstant(
	ctx context.Context,
	queryString string,
	predicate *CollectionRequest,
	flat bool,
) ([]map[string]any, error) {
	timeout := qce.Arguments[metadata.ArgumentKeyTimeout]

	timestamp, err := qce.getComparisonValue(predicate.Timestamp)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), map[string]any{
			"field": metadata.TimestampKey,
		})
	}

	vector, _, err := qce.Client.Query(ctx, queryString, timestamp, timeout)
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
	step := qce.Arguments[metadata.ArgumentKeyStep]
	timeout := qce.Arguments[metadata.ArgumentKeyTimeout]

	start, err := qce.getComparisonValue(predicate.Start)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), map[string]any{
			"field": metadata.TimestampKey,
		})
	}

	end, err := qce.getComparisonValue(predicate.End)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), map[string]any{
			"field": metadata.TimestampKey,
		})
	}

	matrix, _, err := qce.Client.QueryRange(ctx, queryString, start, end, step, timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	sortMatrix(matrix, predicate.OrderBy)
	results := createQueryResultsFromMatrix(matrix, qce.Metric.Labels, qce.Runtime, flat)

	return paginateQueryResults(results, qce.Request.Query), nil
}

func (qce *QueryCollectionExecutor) buildQueryString(
	predicate *CollectionRequest,
) (string, bool, error) {
	if predicate == nil {
		return qce.Request.Collection, true, nil
	}

	conditions := []string{}

	if len(predicate.LabelExpressions) > 0 {
		keys := utils.GetSortedKeys(predicate.LabelExpressions)

		for _, key := range keys {
			expr := predicate.LabelExpressions[key]

			condition, ok, err := (&LabelExpressionBuilder{
				LabelExpression: *expr,
			}).Evaluate(qce.Variables)
			if err != nil || !ok {
				return "", false, err
			}

			conditions = append(conditions, condition)
		}
	}

	valueCondition, err := qce.evalValueComparisonCondition(predicate.Value)
	if err != nil {
		return "", false, err
	}

	query := qce.Request.Collection
	if len(conditions) > 0 {
		query = fmt.Sprintf("%s{%s}", qce.Request.Collection, strings.Join(conditions, ","))
	}

	rawOffset, ok := qce.Arguments[metadata.ArgumentKeyOffset]
	if ok {
		offset, err := client.ParseDuration(rawOffset, qce.Runtime.UnixTimeUnit)
		if err != nil {
			return "", false, fmt.Errorf("invalid offset argument `%v`", rawOffset)
		}

		if offset > 0 {
			query = fmt.Sprintf("%s offset %s", query, offset.String())
		}
	}

	for _, fn := range predicate.Functions {
		query, err = qce.buildQueryStringByFunction(query, fn)
		if err != nil {
			return "", false, err
		}
	}

	query = fmt.Sprintf("%s%s", query, valueCondition)

	return query, true, nil
}

func (qce *QueryCollectionExecutor) evalValueComparisonCondition(
	operator *schema.ExpressionBinaryComparisonOperator,
) (string, error) {
	if operator == nil {
		return "", nil
	}

	v, err := getComparisonValueFloat64(operator.Value, qce.Variables)
	if err != nil {
		return "", fmt.Errorf("invalid value expression: %w", err)
	}

	if v == nil {
		return "", nil
	}

	op, ok := valueBinaryOperators[operator.Operator]
	if !ok {
		return "", fmt.Errorf("value: unsupported comparison operator `%s`", operator)
	}

	return fmt.Sprintf(" %s %f", op, *v), nil
}

func (qce *QueryCollectionExecutor) getComparisonValue(input schema.ComparisonValue) (any, error) {
	return getComparisonValue(input, qce.Variables)
}
