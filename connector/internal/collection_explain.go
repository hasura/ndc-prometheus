package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// QueryCollectionExplainResult holds the result of collection group planning.
type QueryCollectionGroupingExplainResult struct {
	Dimensions       []string          `json:"dimensions"`
	AggregateQueries map[string]string `json:"aggregate_queries"`
}

// QueryCollectionExplainResult holds the result of collection query planning.
type QueryCollectionExplainResult struct {
	OK          bool
	Request     *CollectionRequest
	QueryString string
	Aggregates  map[string]string
	Groups      *QueryCollectionGroupingExplainResult
}

// ToExplainResponse serializes to the explain response.
func (qcer QueryCollectionExplainResult) ToExplainResponse() (*schema.ExplainResponse, error) {
	result := &schema.ExplainResponse{
		Details: schema.ExplainResponseDetails{},
	}

	if qcer.QueryString != "" {
		result.Details["query"] = qcer.QueryString
	}

	for key, agg := range qcer.Aggregates {
		aggKey := fmt.Sprintf("aggregates[%s]", key)
		result.Details[aggKey] = agg
	}

	if qcer.Groups != nil {
		for key, agg := range qcer.Groups.AggregateQueries {
			aggKey := fmt.Sprintf("group_aggregates[%s]", key)
			result.Details[aggKey] = agg
		}
	}

	return result, nil
}

// Explain explains the query request.
func (qce *QueryCollectionExecutor) Explain(
	expressions *CollectionRequest,
) (*QueryCollectionExplainResult, error) {
	collectionQuery := qce.MetricName
	result := &QueryCollectionExplainResult{
		OK:      false,
		Request: expressions,
	}

	if expressions != nil {
		query, ok, err := qce.buildCollectionPredicateQuery(expressions)
		if err != nil {
			return nil, schema.UnprocessableContentError(
				"failed to evaluate the predicate query: "+err.Error(),
				map[string]any{
					"collection": qce.Request.Collection,
				},
			)
		}

		if !ok {
			return result, nil
		}

		collectionQuery = query
	}

	result.OK = true

	collectionQuery, err := qce.buildQueryString(expressions, collectionQuery)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate the query: "+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	result.Aggregates, err = qce.explainAggregates(expressions.Aggregates, collectionQuery)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate aggregates:"+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	result.Groups, err = qce.explainGrouping(expressions.Groups, collectionQuery)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate grouping: "+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	if result.Groups == nil && len(result.Aggregates) == 0 {
		result.QueryString = collectionQuery
	}

	return result, nil
}

// Explain explains the histogram quantile query request with grouping.
func (qce *QueryCollectionExecutor) ExplainHistogramQuantile(
	expressions *CollectionRequest,
) (*QueryCollectionExplainResult, error) {
	result := &QueryCollectionExplainResult{
		OK:      false,
		Request: expressions,
	}

	if expressions == nil {
		expressions = &CollectionRequest{}
	}

	if expressions.Quantile == nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate the predicate query: quantile argument is required",
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	expressions.Functions = []KeyValue{
		{
			Key:   string(metadata.Rate),
			Value: expressions.GetStep(),
		},
	}

	collectionQuery, ok, err := qce.buildCollectionPredicateQuery(expressions)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate the predicate query: "+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	if !ok {
		return result, nil
	}

	result.OK = true

	// build the query string to:
	// rate(hasura_graphql_execution_time_seconds_bucket{...}[$step])
	collectionQuery, err = qce.buildQueryString(expressions, collectionQuery)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate the query: "+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	quantile := *expressions.Quantile
	histogramQuantileFunc := KeyValue{
		Key:   string(metadata.HistogramQuantile),
		Value: quantile,
	}

	if expressions.Groups == nil {
		// generate the histogram quantile query string without grouping.
		// histogram_quantile(
		//     $scalar,
		//     rate(hasura_graphql_execution_time_seconds_bucket{...}[$step])))
		result.QueryString, err = qce.buildQueryStringByFunction(
			expressions,
			collectionQuery,
			histogramQuantileFunc,
		)
		if err != nil {
			return nil, schema.UnprocessableContentError(
				"failed to evaluate histogram quantile: "+err.Error(),
				map[string]any{
					"collection": qce.Request.Collection,
				},
			)
		}

		return result, nil
	}

	return qce.explainHistogramQuantileGrouping(
		expressions,
		result,
		collectionQuery,
		histogramQuantileFunc,
	)
}

func (qce *QueryCollectionExecutor) explainHistogramQuantileGrouping(
	expressions *CollectionRequest,
	result *QueryCollectionExplainResult,
	collectionQuery string,
	histogramQuantileFunc KeyValue,
) (*QueryCollectionExplainResult, error) {
	var err error
	// generate aggregate queries to groups,
	// add the le bucket to grouping
	expressions.Groups.Dimensions = append(expressions.Groups.Dimensions, "le")

	result.Groups, err = qce.explainGrouping(expressions.Groups, collectionQuery)
	if err != nil {
		return nil, schema.UnprocessableContentError(
			"failed to evaluate grouping: "+err.Error(),
			map[string]any{
				"collection": qce.Request.Collection,
			},
		)
	}

	// Finally wrap aggregate queries with histogram_quantile
	for groupKey, groupQuery := range result.Groups.AggregateQueries {
		aggQuery, err := qce.buildQueryStringByFunction(
			expressions,
			groupQuery,
			histogramQuantileFunc,
		)
		if err != nil {
			return nil, schema.UnprocessableContentError(
				"failed to evaluate grouping: "+err.Error(),
				map[string]any{
					"collection": qce.Request.Collection,
				},
			)
		}

		result.Groups.AggregateQueries[groupKey] = aggQuery
	}

	return result, nil
}

func (qce *QueryCollectionExecutor) buildCollectionPredicateQuery(
	predicate *CollectionRequest,
) (string, bool, error) {
	conditions := []string{}

	if len(predicate.LabelExpressions) > 0 {
		keys := utils.GetSortedKeys(predicate.LabelExpressions)

		for _, key := range keys {
			expr := predicate.LabelExpressions[key]

			condition, ok, err := (&LabelExpressionBuilder{
				LabelExpression: *expr,
			}).Evaluate(qce.Variables)
			if err != nil || !ok {
				return "", ok, err
			}

			conditions = append(conditions, condition)
		}
	}

	query := qce.MetricName

	if len(conditions) > 0 {
		query = fmt.Sprintf("%s{%s}", qce.MetricName, strings.Join(conditions, ","))
	}

	if predicate.Offset > 0 && !predicate.HasRangeVectorFunction() {
		query = fmt.Sprintf("%s offset %s", query, predicate.Offset.String())
	}

	return query, true, nil
}

func (qce *QueryCollectionExecutor) buildQueryString(
	predicate *CollectionRequest,
	query string,
) (string, error) {
	valueCondition, err := qce.evalValueComparisonCondition(predicate.Value)
	if err != nil {
		return "", err
	}

	for _, fn := range predicate.Functions {
		query, err = qce.buildQueryStringByFunction(predicate, query, fn)
		if err != nil {
			return "", err
		}
	}

	query = fmt.Sprintf("%s%s", query, valueCondition)

	return query, nil
}

func (qce *QueryCollectionExecutor) buildQueryStringByFunction( //nolint:gocognit,gocyclo,cyclop,funlen,maintidx
	predicate *CollectionRequest,
	query string,
	fn KeyValue,
) (string, error) {
	switch metadata.PromQLFunctionName(fn.Key) {
	case metadata.Absolute,
		metadata.Absent,
		metadata.Ceil,
		metadata.Exponential,
		metadata.Floor,
		metadata.HistogramAvg,
		metadata.HistogramCount,
		metadata.HistogramSum,
		metadata.HistogramStddev,
		metadata.HistogramStdvar,
		metadata.Ln,
		metadata.Log2,
		metadata.Log10,
		metadata.Scalar,
		metadata.Sgn,
		metadata.Sort,
		metadata.SortDesc,
		metadata.Sqrt,
		metadata.Timestamp,
		metadata.Acos,
		metadata.Acosh,
		metadata.Asin,
		metadata.Asinh,
		metadata.Atan,
		metadata.Atanh,
		metadata.Cos,
		metadata.Cosh,
		metadata.Sin,
		metadata.Sinh,
		metadata.Tan,
		metadata.Tanh,
		metadata.Deg,
		metadata.Rad:
		value, err := utils.DecodeNullableBoolean(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
		}

		if value != nil && *value {
			return fmt.Sprintf("%s(%s)", fn.Key, query), nil
		}

		return query, nil
	case metadata.Sum,
		metadata.Avg,
		metadata.Min,
		metadata.Max,
		metadata.Count,
		metadata.Stddev,
		metadata.Stdvar,
		metadata.Group:
		labels, err := utils.DecodeNullableStringSlice(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
		}

		if labels == nil {
			return query, nil
		}

		if len(*labels) == 0 {
			return fmt.Sprintf("%s(%s)", fn.Key, query), nil
		}

		return fmt.Sprintf("%s by (%s) (%s)", fn.Key, strings.Join(*labels, ","), query), nil
	case metadata.SortByLabel, metadata.SortByLabelDesc:
		labels, err := utils.DecodeNullableStringSlice(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
		}

		if labels != nil {
			return fmt.Sprintf(
				"%s(%s%s)",
				fn.Key,
				query,
				buildPromQLParametersFromStringSlice(*labels),
			), nil
		}

		return query, nil
	case metadata.BottomK, metadata.TopK, metadata.LimitK:
		k, err := utils.DecodeNullableInt[int64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid int64 value %v", fn.Key, fn.Value)
		}

		if k != nil {
			return fmt.Sprintf("%s(%d, %s)", fn.Key, *k, query), nil
		}

		return query, nil
	case metadata.Quantile, metadata.LimitRatio, metadata.HistogramQuantile:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
		}

		if n == nil {
			return query, nil
		}

		if *n < 0 || *n > 1 {
			return "", fmt.Errorf(
				"%s: value should be between 0 and 1, got %v",
				fn.Key,
				fn.Value,
			)
		}

		return fmt.Sprintf("%s(%f, %s)", fn.Key, *n, query), nil
	case metadata.Round:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
		}

		if n != nil {
			return fmt.Sprintf("%s(%s, %f)", fn.Key, query, *n), nil
		}

		return query, nil
	case metadata.Clamp:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var boundaryInput ValueBoundaryInput

		if err := mapstructure.Decode(fn.Value, &boundaryInput); err != nil {
			return "", fmt.Errorf("%s: invalid clamp input %w", fn.Key, err)
		}

		return fmt.Sprintf(
			"%s(%s, %f, %f)",
			fn.Key,
			query,
			boundaryInput.Min,
			boundaryInput.Max,
		), nil
	case metadata.HistogramFraction:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var boundary ValueBoundaryInput

		if err := mapstructure.Decode(fn.Value, &boundary); err != nil {
			return "", fmt.Errorf(
				"%s: invalid histogram_fraction input %w",
				fn.Key,
				err,
			)
		}

		return fmt.Sprintf("%s(%f, %f, %s)", fn.Key, boundary.Min, boundary.Max, query), nil
	case metadata.HoltWinters:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var hw HoltWintersInput

		if err := hw.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Key, err)
		}

		return fmt.Sprintf(
			"%s(%s[%s], %f, %f)",
			fn.Key,
			query,
			hw.Range.String(),
			hw.Sf,
			hw.Tf,
		), nil
	case metadata.PredictLinear:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var pli PredictLinearInput

		if err := pli.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Key, err)
		}

		return fmt.Sprintf("%s(%s[%s], %f)", fn.Key, query, pli.Range.String(), pli.T), nil
	case metadata.QuantileOverTime:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var q QuantileOverTimeInput

		if err := q.FromValue(fn.Value, qce.Runtime.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Key, err)
		}

		return fmt.Sprintf("%s(%f, %s[%s])", fn.Key, q.Quantile, query, q.Range.String()), nil
	case metadata.LabelJoin:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var input LabelJoinInput

		if err := mapstructure.Decode(fn.Value, &input); err != nil {
			return "", fmt.Errorf("%s: invalid label_join input %w", fn.Key, err)
		}

		if input.DestLabel == "" {
			return "", fmt.Errorf("%s: the dest_label must not be empty", fn.Key)
		}

		if len(input.SourceLabels) == 0 {
			return "", fmt.Errorf(
				"%s: the source_labels array must have at least 1 item",
				fn.Key,
			)
		}

		return fmt.Sprintf(`%s(%s, %s)`, fn.Key, query, input.String()), nil
	case metadata.LabelReplace:
		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var input LabelReplaceInput

		if err := mapstructure.Decode(fn.Value, &input); err != nil {
			return "", fmt.Errorf("%s: invalid label_join input %w", fn.Key, err)
		}

		if input.DestLabel == "" {
			return "", fmt.Errorf("%s: the dest_label must not be empty", fn.Key)
		}

		return fmt.Sprintf(`%s(%s, %s)`, fn.Key, query, input.String()), nil
	case metadata.ClampMax, metadata.ClampMin:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Key, fn.Value)
		}

		if n != nil {
			return fmt.Sprintf("%s(%s, %f)", fn.Key, query, *n), nil
		}

		return query, nil
	case metadata.CountValues:
		label, err := utils.DecodeNullableString(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Key, fn.Value)
		}

		if label != nil {
			return fmt.Sprintf(`%s("%s", %s)`, fn.Key, *label, query), nil
		}

		return query, nil
	case metadata.AbsentOverTime,
		metadata.Changes,
		metadata.Derivative,
		metadata.Delta,
		metadata.IDelta,
		metadata.Increase,
		metadata.IRate,
		metadata.Rate,
		metadata.Resets,
		metadata.AvgOverTime,
		metadata.MinOverTime,
		metadata.MaxOverTime,
		metadata.MadOverTime,
		metadata.SumOverTime,
		metadata.CountOverTime,
		metadata.StddevOverTime,
		metadata.StdvarOverTime,
		metadata.LastOverTime,
		metadata.PresentOverTime:
		rng, err := qce.Runtime.ParseRangeResolution(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: %w", fn.Key, err)
		}

		if rng != nil {
			var sb strings.Builder

			_, _ = sb.WriteString(fn.Key)
			_, _ = sb.WriteRune('(')
			_, _ = sb.WriteString(query)
			_, _ = sb.WriteRune('[')
			_, _ = sb.WriteString(rng.String())
			_, _ = sb.WriteRune(']')

			if predicate.Offset > 0 && !predicate.OffsetUsed {
				predicate.OffsetUsed = true

				_, _ = sb.WriteString(" offset ")
				_, _ = sb.WriteString(predicate.Offset.String())
			}

			_, _ = sb.WriteRune(')')

			return sb.String(), nil
		}

		return query, nil
	default:
		return "", fmt.Errorf("unsupported promQL function name `%s`", fn.Key)
	}
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

func (qce *QueryCollectionExecutor) explainAggregates(
	aggregates schema.QueryAggregates,
	query string,
) (map[string]string, error) {
	if len(aggregates) == 0 {
		return map[string]string{}, nil
	}

	result := map[string]string{}

	for key, aggregate := range aggregates {
		aggQuery, err := qce.explainGroupingAggregateQuery(query, []string{}, aggregate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}

		result[key] = aggQuery
	}

	return result, nil
}

func (qce *QueryCollectionExecutor) explainGrouping(
	groups *Grouping,
	query string,
) (*QueryCollectionGroupingExplainResult, error) {
	if groups == nil {
		return nil, nil
	}

	result := &QueryCollectionGroupingExplainResult{
		Dimensions:       groups.Dimensions,
		AggregateQueries: make(map[string]string),
	}

	for key, aggregate := range groups.Aggregates {
		aggQuery, err := qce.explainGroupingAggregateQuery(query, groups.Dimensions, aggregate)
		if err != nil {
			return nil, fmt.Errorf("%s: %w", key, err)
		}

		result.AggregateQueries[key] = aggQuery
	}

	return result, nil
}

func (qce *QueryCollectionExecutor) explainGroupingAggregateQuery(
	query string,
	dimensions []string,
	aggregate schema.Aggregate,
) (string, error) {
	aggregateT, err := aggregate.InterfaceT()
	if err != nil {
		return "", err
	}

	switch agg := aggregateT.(type) {
	case *schema.AggregateStarCount:
		var sb strings.Builder

		sb.WriteString("count")

		if len(dimensions) > 0 {
			sb.WriteString(" by (")
			sb.WriteString(strings.Join(dimensions, ", "))
			sb.WriteString(") ")
		}

		sb.WriteRune('(')
		sb.WriteString(query)
		sb.WriteRune(')')

		return sb.String(), nil
	case *schema.AggregateColumnCount:
		if agg.Column == metadata.ValueKey {
			return fmt.Sprintf("count(%s)", query), nil
		}

		return fmt.Sprintf("count by (%s) (%s)", agg.Column, query), nil
	case *schema.AggregateSingleColumn:
		switch agg.Function {
		case string(metadata.Sum), string(metadata.Min), string(metadata.Max), string(metadata.Avg), string(metadata.Stddev), string(metadata.Stdvar):
			if agg.Column != metadata.ValueKey {
				return "", errors.New("support aggregation for the `value` column only")
			}

			var sb strings.Builder

			sb.WriteString(agg.Function)

			if len(dimensions) > 0 {
				sb.WriteString(" by (")
				sb.WriteString(strings.Join(dimensions, ", "))
				sb.WriteString(") ")
			}

			sb.WriteRune('(')
			sb.WriteString(query)
			sb.WriteRune(')')

			return sb.String(), nil
		default:
			return "", fmt.Errorf("unsupported aggregate function: %s", agg.Function)
		}
	default:
		return "", fmt.Errorf("unsupported aggregate type: %s", agg.Type())
	}
}
