package internal

import (
	"encoding/json"
	"fmt"
	"math"
	"slices"
	"strconv"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
)

func createQueryResultsFromVector(vector model.Vector, labels map[string]metadata.LabelInfo, runtime *metadata.RuntimeSettings, flat bool) []map[string]any {
	results := make([]map[string]any, len(vector))
	for i, item := range vector {
		ts := formatTimestamp(item.Timestamp, runtime.Format.Timestamp)
		value := formatValue(item.Value, runtime.Format)
		r := map[string]any{
			metadata.TimestampKey: ts,
			metadata.ValueKey:     value,
			metadata.LabelsKey:    item.Metric,
		}

		for label := range labels {
			r[label] = string(item.Metric[model.LabelName(label)])
		}
		if !flat {
			r[metadata.ValuesKey] = []map[string]any{
				{
					metadata.TimestampKey: ts,
					metadata.ValueKey:     value,
				},
			}
		}

		results[i] = r
	}

	return results
}

func createQueryResultsFromMatrix(matrix model.Matrix, labels map[string]metadata.LabelInfo, runtime *metadata.RuntimeSettings, flat bool) []map[string]any {
	if flat {
		return createFlatQueryResultsFromMatrix(matrix, labels, runtime)
	}

	return createGroupQueryResultsFromMatrix(matrix, labels, runtime)
}

func createGroupQueryResultsFromMatrix(matrix model.Matrix, labels map[string]metadata.LabelInfo, runtime *metadata.RuntimeSettings) []map[string]any {
	results := make([]map[string]any, len(matrix))
	for i, item := range matrix {
		r := map[string]any{
			metadata.LabelsKey: item.Metric,
		}

		for label := range labels {
			r[label] = string(item.Metric[model.LabelName(label)])
		}

		valuesLen := len(item.Values)
		values := make([]map[string]any, valuesLen)
		for i, value := range item.Values {
			ts := formatTimestamp(value.Timestamp, runtime.Format.Timestamp)
			v := formatValue(value.Value, runtime.Format)
			values[i] = map[string]any{
				metadata.TimestampKey: ts,
				metadata.ValueKey:     v,
			}
			if i == valuesLen-1 {
				r[metadata.TimestampKey] = ts
				r[metadata.ValueKey] = v
			}
		}

		r[metadata.ValuesKey] = values
		results[i] = r
	}

	return results
}

func createFlatQueryResultsFromMatrix(matrix model.Matrix, labels map[string]metadata.LabelInfo, runtime *metadata.RuntimeSettings) []map[string]any {
	results := []map[string]any{}

	for _, item := range matrix {
		for _, value := range item.Values {
			ts := formatTimestamp(value.Timestamp, runtime.Format.Timestamp)
			v := formatValue(value.Value, runtime.Format)
			r := map[string]any{
				metadata.LabelsKey:    item.Metric,
				metadata.TimestampKey: ts,
				metadata.ValueKey:     v,
				metadata.ValuesKey:    nil,
			}

			for label := range labels {
				r[label] = string(item.Metric[model.LabelName(label)])
			}

			results = append(results, r)
		}
	}

	return results
}

func formatTimestamp(ts model.Time, format metadata.TimestampFormat) any {
	switch format {
	case metadata.TimestampUnix:
		return ts.Unix()
	case metadata.TimestampUnixMilli:
		return ts.Time().UnixMilli()
	case metadata.TimestampUnixMicro:
		return ts.Time().UnixMicro()
	case metadata.TimestampUnixNano:
		return strconv.FormatInt(ts.UnixNano(), 10)
	default:
		return ts.Time().Format(time.RFC3339)
	}
}

func formatValue(value model.SampleValue, format metadata.RuntimeFormatSettings) any {
	switch format.Value {
	case metadata.ValueFloat64:
		if math.IsNaN(float64(value)) {
			return format.NaN
		}
		if value > 0 && math.IsInf(float64(value), 1) {
			return format.Inf
		}
		if value < 0 && math.IsInf(float64(value), -1) {
			return format.NegativeInf
		}

		return float64(value)
	default:
		return value.String()
	}
}

func decodeStringSlice(value any) ([]string, error) {
	if utils.IsNil(value) {
		return nil, nil
	}
	var err error
	sliceValue := []string{}
	if str, ok := value.(string); ok {
		// try to parse the slice from the json string
		err = json.Unmarshal([]byte(str), &sliceValue)
	} else {
		sliceValue, err = utils.DecodeStringSlice(value)
	}
	if err != nil {
		return nil, err
	}

	return sliceValue, nil
}

func intersection[T comparable](sliceA []T, sliceB []T) []T {
	var result []T
	if len(sliceA) == 0 || len(sliceB) == 0 {
		return result
	}

	for _, a := range sliceA {
		if slices.Contains(sliceB, a) {
			result = append(result, a)
		}
	}

	return result
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

func getComparisonValueFloat64(input schema.ComparisonValue, variables map[string]any) (*float64, error) {
	rawValue, err := getComparisonValue(input, variables)
	if err != nil {
		return nil, err
	}
	return utils.DecodeNullableFloat[float64](rawValue)
}

func getComparisonValueString(input schema.ComparisonValue, variables map[string]any) (*string, error) {
	rawValue, err := getComparisonValue(input, variables)
	if err != nil {
		return nil, err
	}
	return utils.DecodeNullableString(rawValue)
}

func getComparisonValueStringSlice(input schema.ComparisonValue, variables map[string]any) ([]string, error) {
	rawValue, err := getComparisonValue(input, variables)
	if err != nil {
		return nil, err
	}
	return decodeStringSlice(rawValue)
}
