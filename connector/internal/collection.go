package internal

import (
	"context"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
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
	Client    *client.Client
	Tracer    trace.Tracer
	Runtime   *metadata.RuntimeSettings
	Request   *schema.QueryRequest
	Metric    metadata.MetricInfo
	Variables map[string]any
	Arguments map[string]any
}

// Explain explains the query request
func (qce *QueryCollectionExecutor) Explain(ctx context.Context) (*CollectionRequest, string, bool, error) {
	expressions, err := EvalCollectionRequest(qce.Request, qce.Arguments)
	if err != nil {
		return nil, "", false, schema.UnprocessableContentError(err.Error(), map[string]any{
			"collection": qce.Request.Collection,
		})
	}

	queryString, ok, err := qce.buildQueryString(expressions)
	if err != nil {
		return nil, "", false, schema.UnprocessableContentError("failed to evaluate the query", map[string]any{
			"collection": qce.Request.Collection,
			"error":      err.Error(),
		})
	}

	return expressions, queryString, ok, nil
}

// Execute executes the query request
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
		return nil, schema.UnprocessableContentError(fmt.Sprintf("expected boolean type for the flat field, got: %v", err), map[string]any{
			"field": metadata.ArgumentKeyFlat,
		})
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

func (qce *QueryCollectionExecutor) queryInstant(ctx context.Context, queryString string, predicate *CollectionRequest, flat bool) ([]map[string]any, error) {
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

	qce.sortVector(vector, predicate.OrderBy)

	if qce.Request.Query.Offset != nil && *qce.Request.Query.Offset > 0 {
		if len(vector) <= *qce.Request.Query.Offset {
			return []map[string]any{}, nil
		}
		vector = vector[*qce.Request.Query.Offset:]
	}
	if qce.Request.Query.Limit != nil && *qce.Request.Query.Limit < len(vector) {
		vector = vector[:*qce.Request.Query.Limit]
	}

	results := createQueryResultsFromVector(vector, qce.Metric.Labels, qce.Runtime, flat)
	return results, nil
}

func (qce *QueryCollectionExecutor) queryRange(ctx context.Context, queryString string, predicate *CollectionRequest, flat bool) ([]map[string]any, error) {
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

	qce.sortMatrix(matrix, predicate.OrderBy)

	if qce.Request.Query.Offset != nil && *qce.Request.Query.Offset > 0 {
		if len(matrix) <= *qce.Request.Query.Offset {
			return []map[string]any{}, nil
		}
		matrix = matrix[*qce.Request.Query.Offset:]
	}
	results := createQueryResultsFromMatrix(matrix, qce.Metric.Labels, qce.Runtime, flat)

	if qce.Request.Query.Limit != nil && *qce.Request.Query.Limit < len(results) {
		results = results[:*qce.Request.Query.Limit]
	}

	return results, nil
}

func (qce *QueryCollectionExecutor) buildQueryString(predicate *CollectionRequest) (string, bool, error) {

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
		switch metadata.PromQLFunctionName(fn.Key) {
		case metadata.Absolute, metadata.Absent, metadata.Ceil, metadata.Exponential, metadata.Floor, metadata.HistogramAvg, metadata.HistogramCount, metadata.HistogramSum, metadata.HistogramStddev, metadata.HistogramStdvar, metadata.Ln, metadata.Log2, metadata.Log10, metadata.Scalar, metadata.Sgn, metadata.Sort, metadata.SortDesc, metadata.Sqrt, metadata.Timestamp, metadata.Acos, metadata.Acosh, metadata.Asin, metadata.Asinh, metadata.Atan, metadata.Atanh, metadata.Cos, metadata.Cosh, metadata.Sin, metadata.Sinh, metadata.Tan, metadata.Tanh, metadata.Deg, metadata.Rad:
			value, err := utils.DecodeNullableBoolean(fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
			}
			if value != nil && *value {
				query = fmt.Sprintf("%s(%s)", fn.Key, query)
			}
		case metadata.Sum, metadata.Avg, metadata.Min, metadata.Max, metadata.Count, metadata.Stddev, metadata.Stdvar, metadata.Group:
			labels, err := utils.DecodeNullableStringSlice(fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
			}
			if labels != nil {
				if len(*labels) == 0 {
					query = fmt.Sprintf("%s(%s)", fn.Key, query)
				} else {
					query = fmt.Sprintf("%s by (%s) (%s)", fn.Key, strings.Join(*labels, ","), query)
				}
			}
		case metadata.SortByLabel, metadata.SortByLabelDesc:
			labels, err := utils.DecodeNullableStringSlice(fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
			}
			if labels != nil {
				query = fmt.Sprintf("%s(%s%s)", fn.Key, query, buildPromQLParametersFromStringSlice(*labels))
			}
		case metadata.BottomK, metadata.TopK, metadata.LimitK:
			k, err := utils.DecodeNullableInt[int64](fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid int64 value %v", fn.Key, fn.Value)
			}
			if k != nil {
				query = fmt.Sprintf("%s(%d, %s)", fn.Key, *k, query)
			}
		case metadata.Quantile, metadata.LimitRatio, metadata.HistogramQuantile:
			n, err := utils.DecodeNullableFloat[float64](fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
			}
			if n != nil {
				if *n < 0 || *n > 1 {
					return "", false, fmt.Errorf("%s: value should be between 0 and 1, got %v", fn.Key, fn.Value)
				}
				query = fmt.Sprintf("%s(%f, %s)", fn.Key, *n, query)
			}
		case metadata.Round:
			n, err := utils.DecodeNullableFloat[float64](fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
			}
			if n != nil {
				query = fmt.Sprintf("%s(%s, %f)", fn.Key, query, *n)
			}
		case metadata.Clamp:
			if !utils.IsNil(fn.Value) {
				var boundaryInput ValueBoundaryInput
				if err := mapstructure.Decode(fn.Value, &boundaryInput); err != nil {
					return "", false, fmt.Errorf("%s: invalid clamp input %s", fn.Key, err)
				}
				query = fmt.Sprintf("%s(%s, %f, %f)", fn.Key, query, boundaryInput.Min, boundaryInput.Max)
			}
		case metadata.HistogramFraction:
			if !utils.IsNil(fn.Value) {
				var boundary ValueBoundaryInput
				if err := mapstructure.Decode(fn.Value, &boundary); err != nil {
					return "", false, fmt.Errorf("%s: invalid histogram_fraction input %s", fn.Key, err)
				}
				query = fmt.Sprintf("%s(%f, %f, %s)", fn.Key, boundary.Min, boundary.Max, query)
			}
		case metadata.HoltWinters:
			if !utils.IsNil(fn.Value) {
				var hw HoltWintersInput
				if err := hw.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
					return "", false, fmt.Errorf("%s: %s", fn.Key, err)
				}
				query = fmt.Sprintf("%s(%s[%s], %f, %f)", fn.Key, query, hw.Range.String(), hw.Sf, hw.Tf)
			}
		case metadata.PredictLinear:
			if !utils.IsNil(fn.Value) {
				var pli PredictLinearInput
				if err := pli.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
					return "", false, fmt.Errorf("%s: %s", fn.Key, err)
				}
				query = fmt.Sprintf("%s(%s[%s], %f)", fn.Key, query, pli.Range.String(), pli.T)
			}
		case metadata.QuantileOverTime:
			if !utils.IsNil(fn.Value) {
				var q QuantileOverTimeInput
				if err := q.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
					return "", false, fmt.Errorf("%s: %s", fn.Key, err)
				}
				query = fmt.Sprintf("%s(%f, %s[%s])", fn.Key, q.Quantile, query, q.Range.String())
			}
		case metadata.LabelJoin:
			if !utils.IsNil(fn.Value) {
				var input LabelJoinInput
				if err := mapstructure.Decode(fn.Value, &input); err != nil {
					return "", false, fmt.Errorf("%s: invalid label_join input %s", fn.Key, err)
				}
				if input.DestLabel == "" {
					return "", false, fmt.Errorf("%s: the dest_label must not be empty", fn.Key)
				}
				if len(input.SourceLabels) == 0 {
					return "", false, fmt.Errorf("%s: the source_labels array must have at least 1 item", fn.Key)
				}
				query = fmt.Sprintf(`%s(%s, %s)`, fn.Key, query, input.String())
			}
		case metadata.LabelReplace:
			if !utils.IsNil(fn.Value) {
				var input LabelReplaceInput
				if err := mapstructure.Decode(fn.Value, &input); err != nil {
					return "", false, fmt.Errorf("%s: invalid label_join input %s", fn.Key, err)
				}
				if input.DestLabel == "" {
					return "", false, fmt.Errorf("%s: the dest_label must not be empty", fn.Key)
				}
				query = fmt.Sprintf(`%s(%s, %s)`, fn.Key, query, input.String())
			}

		case metadata.ClampMax, metadata.ClampMin:
			n, err := utils.DecodeNullableFloat[float64](fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
			}
			if n != nil {
				query = fmt.Sprintf("%s(%s, %f)", fn.Key, query, *n)
			}
		case metadata.CountValues:
			label, err := utils.DecodeNullableString(fn.Value)
			if err != nil {
				return "", false, fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
			}
			if label != nil {
				query = fmt.Sprintf(`%s("%s", %s)`, fn.Key, *label, query)
			}
		case metadata.AbsentOverTime, metadata.Changes, metadata.Derivative, metadata.Delta, metadata.IDelta, metadata.Increase, metadata.IRate, metadata.Rate, metadata.Resets, metadata.AvgOverTime, metadata.MinOverTime, metadata.MaxOverTime, metadata.MadOverTime, metadata.SumOverTime, metadata.CountOverTime, metadata.StddevOverTime, metadata.StdvarOverTime, metadata.LastOverTime, metadata.PresentOverTime:
			rng, err := client.ParseRangeResolution(fn.Value, qce.Runtime.UnixTimeUnit)
			if err != nil {
				return "", false, fmt.Errorf("%s: %s", fn.Key, err)
			}
			if rng != nil {
				query = fmt.Sprintf(`%s(%s[%s])`, fn.Key, query, rng.String())
			}
		default:
			return "", false, fmt.Errorf("unsupported promQL function name `%s`", fn.Key)
		}
	}

	query = fmt.Sprintf("%s%s", query, valueCondition)

	return query, true, nil
}

func (qce *QueryCollectionExecutor) evalValueComparisonCondition(operator *schema.ExpressionBinaryComparisonOperator) (string, error) {
	if operator == nil {
		return "", nil
	}
	value, err := qce.getComparisonValue(operator.Value)
	if err != nil {
		return "", fmt.Errorf("invalid value expression: %s", err)
	}
	v, err := utils.DecodeNullableFloat[float64](value)
	if err != nil {
		return "", fmt.Errorf("invalid value expression: %s", err)
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

func getComparisonValue(input schema.ComparisonValue, variables map[string]any) (any, error) {
	if len(input) == 0 {
		return nil, nil
	}

	switch v := input.Interface().(type) {
	case *schema.ComparisonValueScalar:
		return v.Value, nil
	case *schema.ComparisonValueVariable:
		if value, ok := variables[v.Name]; ok {
			return value, nil
		}
		return nil, fmt.Errorf("variable %s does not exist", v.Name)
	default:
		return nil, fmt.Errorf("invalid comparison value: %v", input)
	}
}
