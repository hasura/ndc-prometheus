package internal

import (
	"fmt"
	"strings"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/utils"
)

// PromQLFunction represents a promql function.
type PromQLFunction struct {
	Name  metadata.PromQLFunctionName
	Value any
}

// Render builds a promQL query string from the function.
func (fn *PromQLFunction) Render( //nolint:gocognit,gocyclo,cyclop,funlen,maintidx
	query string,
	offset time.Duration,
	settings *metadata.RuntimeSettings,
) (string, error) {
	var strOffset string

	if offset > 0 {
		strOffset = " offset " + offset.String()
	}

	switch metadata.PromQLFunctionName(fn.Name) {
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
			return "", fmt.Errorf("%s: invalid value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if value != nil && *value {
			return fmt.Sprintf("%s(%s)", fn.Name, query), nil
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
			return "", fmt.Errorf("%s: invalid value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if labels == nil {
			return query, nil
		}

		if len(*labels) == 0 {
			return fmt.Sprintf("%s(%s)", fn.Name, query), nil
		}

		return fmt.Sprintf("%s by (%s) (%s)", fn.Name, strings.Join(*labels, ","), query), nil
	case metadata.SortByLabel, metadata.SortByLabelDesc:
		labels, err := utils.DecodeNullableStringSlice(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if labels != nil {
			return fmt.Sprintf(
				"%s(%s%s)",
				fn.Name,
				query,
				buildPromQLParametersFromStringSlice(*labels),
			), nil
		}

		return query, nil
	case metadata.BottomK, metadata.TopK, metadata.LimitK:
		k, err := utils.DecodeNullableInt[int64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid int64 value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if k != nil {
			return fmt.Sprintf("%s(%d, %s)", fn.Name, *k, query), nil
		}

		return query, nil
	case metadata.Quantile, metadata.LimitRatio, metadata.HistogramQuantile:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if n == nil {
			return query, nil
		}

		if *n < 0 || *n > 1 {
			return "", fmt.Errorf(
				"%s: value should be between 0 and 1, got %v",
				fn.Name,
				fn.Value,
			)
		}

		return fmt.Sprintf("%s(%f, %s)", fn.Name, *n, query), nil
	case metadata.Round:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if n != nil {
			return fmt.Sprintf("%s(%s, %f)", fn.Name, query, *n), nil
		}

		return query, nil
	case metadata.Clamp:
		query += strOffset

		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var boundaryInput ValueBoundaryInput

		if err := mapstructure.Decode(fn.Value, &boundaryInput); err != nil {
			return "", fmt.Errorf("%s: invalid clamp input %w", fn.Name, err)
		}

		return fmt.Sprintf(
			"%s(%s, %f, %f)",
			fn.Name,
			query,
			boundaryInput.Min,
			boundaryInput.Max,
		), nil
	case metadata.HistogramFraction:
		query += strOffset

		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var boundary ValueBoundaryInput

		if err := mapstructure.Decode(fn.Value, &boundary); err != nil {
			return "", fmt.Errorf(
				"%s: invalid histogram_fraction input %w",
				fn.Name,
				err,
			)
		}

		return fmt.Sprintf("%s(%f, %f, %s)", fn.Name, boundary.Min, boundary.Max, query), nil
	case metadata.HoltWinters:
		if utils.IsNil(fn.Value) {
			return query + strOffset, nil
		}

		var hw HoltWintersInput

		if err := hw.FromValue(fn.Value, settings.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Name, err)
		}

		return fmt.Sprintf(
			"%s(%s[%s]%s, %f, %f)",
			fn.Name,
			query,
			hw.Range.String(),
			strOffset,
			hw.Sf,
			hw.Tf,
		), nil
	case metadata.PredictLinear:
		if utils.IsNil(fn.Value) {
			return query + strOffset, nil
		}

		var pli PredictLinearInput

		if err := pli.FromValue(fn.Value, settings.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Name, err)
		}

		return fmt.Sprintf("%s(%s[%s]%s, %f)", fn.Name, query, strOffset, pli.Range.String(), pli.T), nil
	case metadata.QuantileOverTime:
		if utils.IsNil(fn.Value) {
			return query + strOffset, nil
		}

		var q QuantileOverTimeInput

		if err := q.FromValue(fn.Value, settings.UnixTimeUnit); err != nil {
			return "", fmt.Errorf("%s: %w", fn.Name, err)
		}

		return fmt.Sprintf("%s(%f, %s[%s]%s)", fn.Name, q.Quantile, query, strOffset, q.Range.String()), nil
	case metadata.LabelJoin:
		query += strOffset

		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var input LabelJoinInput

		if err := mapstructure.Decode(fn.Value, &input); err != nil {
			return "", fmt.Errorf("%s: invalid label_join input %w", fn.Name, err)
		}

		if input.DestLabel == "" {
			return "", fmt.Errorf("%s: the dest_label must not be empty", fn.Name)
		}

		if len(input.SourceLabels) == 0 {
			return "", fmt.Errorf(
				"%s: the source_labels array must have at least 1 item",
				fn.Name,
			)
		}

		return fmt.Sprintf(`%s(%s, %s)`, fn.Name, query, input.String()), nil
	case metadata.LabelReplace:
		query += strOffset

		if utils.IsNil(fn.Value) {
			return query, nil
		}

		var input LabelReplaceInput

		if err := mapstructure.Decode(fn.Value, &input); err != nil {
			return "", fmt.Errorf("%s: invalid label_join input %w", fn.Name, err)
		}

		if input.DestLabel == "" {
			return "", fmt.Errorf("%s: the dest_label must not be empty", fn.Name)
		}

		return fmt.Sprintf(`%s(%s, %s)`, fn.Name, query, input.String()), nil
	case metadata.ClampMax, metadata.ClampMin:
		n, err := utils.DecodeNullableFloat[float64](fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid float64 value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if n != nil {
			return fmt.Sprintf("%s(%s, %f)", fn.Name, query, *n), nil
		}

		return query, nil
	case metadata.CountValues:
		label, err := utils.DecodeNullableString(fn.Value)
		if err != nil {
			return "", fmt.Errorf("%s: invalid value %v", fn.Name, fn.Value)
		}

		query += strOffset

		if label != nil {
			return fmt.Sprintf(`%s("%s", %s)`, fn.Name, *label, query), nil
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
		rng, err := client.ParseRangeResolution(fn.Value, settings.UnixTimeUnit)
		if err != nil {
			return "", fmt.Errorf("%s: %w", fn.Name, err)
		}

		if rng == nil {
			return query + strOffset, nil
		}

		return fmt.Sprintf(`%s(%s[%s]%s)`, fn.Name, query, rng.String(), strOffset), nil
	default:
		return "", fmt.Errorf("unsupported promQL function name `%s`", fn.Name)
	}
}
