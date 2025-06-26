package internal

import (
	"context"
	"fmt"
	"regexp"
	"slices"
	"strconv"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/trace"
)

type NativeQueryExecutor struct {
	Client      *client.Client
	Tracer      trace.Tracer
	Runtime     *metadata.RuntimeSettings
	Request     *schema.QueryRequest
	NativeQuery *metadata.NativeQuery
	Arguments   map[string]any
	Variables   map[string]any
}

// Explain explains the query request.
func (nqe *NativeQueryExecutor) Explain(ctx context.Context) (*NativeQueryRequest, string, error) {
	params, err := EvalNativeQueryRequest(nqe.Request, nqe.Arguments, nqe.Variables, nqe.Runtime)
	if err != nil {
		return nil, "", err
	}

	queryString, err := nqe.evalArguments(params)
	if err != nil {
		return nil, "", err
	}

	if unresolvedArguments := metadata.FindNativeQueryVariableNames(queryString); len(
		unresolvedArguments,
	) > 0 {
		return nil, "", schema.UnprocessableContentError(
			fmt.Sprintf("unresolved variables %v in the query", unresolvedArguments),
			map[string]any{
				"collection": nqe.Request.Collection,
				"query":      queryString,
			},
		)
	}

	return params, queryString, nil
}

func (nqe *NativeQueryExecutor) evalArguments(params *NativeQueryRequest) (string, error) {
	var step time.Duration
	var err error

	queryString := nqe.NativeQuery.Query

	for key, arg := range nqe.Arguments {
		switch key {
		case metadata.ArgumentKeyStep:
			step, err = nqe.Runtime.ParseDuration(arg)
			if err != nil {
				return "", schema.UnprocessableContentError(err.Error(), nil)
			}
		case metadata.ArgumentKeyTimeout:
			params.Timeout, err = nqe.Runtime.ParseDuration(arg)
			if err != nil {
				return "", schema.UnprocessableContentError(err.Error(), nil)
			}
		default:
			argInfo, ok := nqe.NativeQuery.Arguments[key]
			if !ok {
				break
			}

			var argString string

			switch metadata.ScalarName(argInfo.Type) {
			case metadata.ScalarInt64:
				argInt, err := utils.DecodeInt[int64](arg)
				if err != nil {
					return "", schema.UnprocessableContentError(err.Error(), nil)
				}

				argString = strconv.FormatInt(argInt, 10)
			case metadata.ScalarFloat64:
				argFloat, err := utils.DecodeFloat[float64](arg)
				if err != nil {
					return "", schema.UnprocessableContentError(err.Error(), nil)
				}

				argString = fmt.Sprint(argFloat)
			case metadata.ScalarDuration:
				duration, err := nqe.Runtime.ParseRangeResolution(arg)
				if err != nil {
					return "", schema.UnprocessableContentError(err.Error(), nil)
				}

				if duration == nil {
					return "", schema.UnprocessableContentError(
						fmt.Sprintf("argument `%s` is required", key),
						nil,
					)
				}

				argString = duration.String()
			default:
				argString, err = utils.DecodeString(arg)
				if err != nil {
					return "", schema.UnprocessableContentError(
						fmt.Sprintf("%s: %s", key, err.Error()),
						nil,
					)
				}
			}

			queryString = metadata.ReplaceNativeQueryVariable(queryString, key, argString)
		}
	}

	if params.start != nil || params.end != nil {
		params.Range, err = metadata.NewRange(params.start, params.end, step)
		if err != nil {
			return "", schema.UnprocessableContentError(err.Error(), nil)
		}
	}

	return queryString, nil
}

// Execute executes the query request.
func (nqe *NativeQueryExecutor) Execute(ctx context.Context) (*schema.RowSet, error) {
	ctx, span := nqe.Tracer.Start(ctx, "Execute Native Query")
	defer span.End()

	params, queryString, err := nqe.Explain(ctx)
	if err != nil {
		return nil, err
	}

	return nqe.execute(ctx, params, queryString)
}

func (nqe *NativeQueryExecutor) execute(
	ctx context.Context,
	params *NativeQueryRequest,
	queryString string,
) (*schema.RowSet, error) {
	nullableFlat, err := utils.DecodeNullableBoolean(nqe.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(
			fmt.Sprintf("expected boolean type for the flat field, got: %v", err),
			map[string]any{
				"field": metadata.ArgumentKeyFlat,
			},
		)
	}

	var rawResults []map[string]any
	flat := nqe.Runtime.IsFlat(nullableFlat)

	if !utils.IsNil(params.Timestamp) {
		rawResults, err = nqe.queryInstant(ctx, queryString, params, flat)
	} else {
		rawResults, err = nqe.queryRange(ctx, queryString, params, flat)
	}

	if err != nil {
		return nil, err
	}

	results, err := utils.EvalObjectsWithColumnSelection(nqe.Request.Query.Fields, rawResults)
	if err != nil {
		return nil, err
	}

	return &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
		Rows:       results,
	}, nil
}

func (nqe *NativeQueryExecutor) queryInstant(
	ctx context.Context,
	queryString string,
	params *NativeQueryRequest,
	flat bool,
) ([]map[string]any, error) {
	vector, _, err := nqe.Client.Query(ctx, queryString, params.Timestamp, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent("post_filter", trace.WithAttributes(
		utils.JSONAttribute("expression", params.Expression),
		attribute.Int("pre_filter_count", len(vector)),
	))

	vector, err = nqe.filterVectorResults(vector, params.Expression)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	span.AddEvent(
		"post_filter_results",
		trace.WithAttributes(attribute.Int("post_filter_count", len(vector))),
	)
	sortVector(vector, params.OrderBy)
	vector = paginateVector(vector, nqe.Request.Query)
	results := createQueryResultsFromVector(vector, nqe.NativeQuery.Labels, nqe.Runtime, flat)

	return results, nil
}

func (nqe *NativeQueryExecutor) queryRange(
	ctx context.Context,
	queryString string,
	params *NativeQueryRequest,
	flat bool,
) ([]map[string]any, error) {
	matrix, _, err := nqe.Client.QueryRange(
		ctx,
		queryString,
		*params.Range,
		params.Timeout,
	)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	span := trace.SpanFromContext(ctx)
	span.AddEvent("post_filter", trace.WithAttributes(
		utils.JSONAttribute("expression", params.Expression),
		attribute.Int("pre_filter_count", len(matrix)),
	))

	matrix, err = nqe.filterMatrixResults(matrix, params)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	span.AddEvent(
		"post_filter_results",
		trace.WithAttributes(attribute.Int("post_filter_count", len(matrix))),
	)
	sortMatrix(matrix, params.OrderBy)
	results := createQueryResultsFromMatrix(matrix, nqe.NativeQuery.Labels, nqe.Runtime, flat)

	return paginateQueryResults(results, nqe.Request.Query), nil
}

func (nqe *NativeQueryExecutor) filterVectorResults(
	vector model.Vector,
	expr schema.Expression,
) (model.Vector, error) {
	if expr == nil || len(vector) == 0 {
		return vector, nil
	}

	results := model.Vector{}

	for _, item := range vector {
		valid, err := nqe.validateBoolExp(item.Metric, item.Value, expr)
		if err != nil {
			return nil, err
		}

		if valid {
			results = append(results, item)
		}
	}

	return results, nil
}

func (nqe *NativeQueryExecutor) filterMatrixResults(
	matrix model.Matrix,
	params *NativeQueryRequest,
) (model.Matrix, error) {
	if params.Expression == nil || len(matrix) == 0 {
		return matrix, nil
	}

	results := model.Matrix{}

	for _, item := range matrix {
		if !params.HasValueBoolExp {
			valid, err := nqe.validateBoolExp(item.Metric, 0, params.Expression)
			if err != nil {
				return nil, err
			}

			if valid {
				results = append(results, item)
			}

			continue
		}

		newItem := model.SampleStream{
			Metric:     item.Metric,
			Histograms: item.Histograms,
		}

		for _, v := range item.Values {
			valid, err := nqe.validateBoolExp(item.Metric, v.Value, params.Expression)
			if err != nil {
				return nil, err
			}

			if valid {
				newItem.Values = append(newItem.Values, v)
			}
		}

		if len(newItem.Values) > 0 {
			results = append(results, &newItem)
		}
	}

	return results, nil
}

func (nqe *NativeQueryExecutor) validateBoolExp(
	labels model.Metric,
	value model.SampleValue,
	expr schema.Expression,
) (bool, error) {
	switch exprs := expr.Interface().(type) {
	case *schema.ExpressionAnd:
		for _, e := range exprs.Expressions {
			valid, err := nqe.validateBoolExp(labels, value, e)
			if !valid || err != nil {
				return false, err
			}
		}

		return true, nil
	case *schema.ExpressionBinaryComparisonOperator:
		return nqe.validateExpressionBinaryComparisonOperator(labels, value, exprs)
	case *schema.ExpressionNot:
		valid, err := nqe.validateBoolExp(labels, value, exprs.Expression)
		if err != nil {
			return false, err
		}

		return !valid, nil
	case *schema.ExpressionOr:
		if len(exprs.Expressions) == 0 {
			return true, nil
		}

		for _, e := range exprs.Expressions {
			valid, err := nqe.validateBoolExp(labels, value, e)
			if err != nil {
				return false, err
			}

			if valid {
				return true, nil
			}
		}

		return false, nil
	case *schema.ExpressionUnaryComparisonOperator:
		return nqe.validateExpressionUnaryComparisonOperator(labels, exprs)
	default:
		return false, fmt.Errorf("unsupported expression %v", expr)
	}
}

func (nqe *NativeQueryExecutor) validateExpressionBinaryComparisonOperator(
	labels model.Metric,
	value model.SampleValue,
	exprs *schema.ExpressionBinaryComparisonOperator,
) (bool, error) {
	targetT, err := exprs.Column.InterfaceT()
	if err != nil {
		return false, err
	}

	switch target := targetT.(type) {
	case *schema.ComparisonTargetColumn:
		return nqe.validateExpressionBinaryComparisonColumn(labels, value, exprs, target)
	default:
		return false, fmt.Errorf("unsupported operator %s", targetT.Type())
	}
}

func (nqe *NativeQueryExecutor) validateExpressionBinaryComparisonColumn(
	labels model.Metric,
	value model.SampleValue,
	exprs *schema.ExpressionBinaryComparisonOperator,
	target *schema.ComparisonTargetColumn,
) (bool, error) {
	if target.Name == metadata.ValueKey {
		return nqe.validateExpressionBinaryComparisonColumnValue(value, exprs)
	}

	labelValue := labels[model.LabelName(target.Name)]

	switch exprs.Operator {
	case metadata.Equal, metadata.NotEqual, metadata.Regex, metadata.NotRegex:
		value, err := getComparisonValueString(exprs.Value, nqe.Variables)
		if err != nil {
			return false, err
		}

		if value == nil {
			return true, nil
		}

		if exprs.Operator == metadata.Equal {
			return *value == string(labelValue), nil
		}

		if exprs.Operator == metadata.NotEqual {
			return *value != string(labelValue), nil
		}

		regex, err := regexp.Compile(*value)
		if err != nil {
			return false, fmt.Errorf("invalid regular expression %s: %w", *value, err)
		}

		if exprs.Operator == metadata.Regex {
			return regex.MatchString(string(labelValue)), nil
		}

		return !regex.MatchString(string(labelValue)), nil
	case metadata.In, metadata.NotIn:
		value, err := getComparisonValueStringSlice(exprs.Value, nqe.Variables)
		if err != nil {
			return false, fmt.Errorf("failed to decode string array; %w", err)
		}

		if value == nil {
			return true, nil
		}

		if exprs.Operator == metadata.In {
			return slices.Contains(value, string(labelValue)), nil
		} else {
			return !slices.Contains(value, string(labelValue)), nil
		}
	}

	return false, nil
}

func (nqe *NativeQueryExecutor) validateExpressionBinaryComparisonColumnValue(
	value model.SampleValue,
	exprs *schema.ExpressionBinaryComparisonOperator,
) (bool, error) {
	floatValue, err := getComparisonValueFloat64(exprs.Value, nqe.Variables)
	if err != nil {
		return false, err
	}

	if floatValue == nil {
		return true, nil
	}

	switch exprs.Operator {
	case metadata.Equal:
		return value.Equal(model.SampleValue(*floatValue)), nil
	case metadata.NotEqual:
		return !value.Equal(model.SampleValue(*floatValue)), nil
	case metadata.Least:
		return float64(value) < *floatValue, nil
	case metadata.LeastOrEqual:
		return float64(value) <= *floatValue, nil
	case metadata.Greater:
		return float64(value) > *floatValue, nil
	case metadata.GreaterOrEqual:
		return float64(value) >= *floatValue, nil
	default:
		return false, fmt.Errorf("unsupported value operator %s", exprs.Operator)
	}
}

func (nqe *NativeQueryExecutor) validateExpressionUnaryComparisonOperator(
	labels model.Metric,
	exprs *schema.ExpressionUnaryComparisonOperator,
) (bool, error) {
	switch exprs.Operator {
	case schema.UnaryComparisonOperatorIsNull:
		targetT, err := exprs.Column.InterfaceT()
		if err != nil {
			return false, err
		}

		switch target := targetT.(type) {
		case *schema.ComparisonTargetColumn:
			_, ok := labels[model.LabelName(target.Name)]

			return !ok, nil
		default:
			return false, fmt.Errorf("unsupported comparison target %s", targetT.Type())
		}
	default:
		return false, fmt.Errorf("unsupported comparison operator %s", exprs.Operator)
	}
}
