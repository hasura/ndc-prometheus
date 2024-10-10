package internal

import (
	"encoding/json"
	"slices"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
)

func createQueryResultsFromVector(vector model.Vector, labels map[string]metadata.LabelInfo, runtime *metadata.RuntimeSettings, flat bool) []map[string]any {
	results := make([]map[string]any, len(vector))
	for i, item := range vector {
		ts := formatTimestamp(item.Timestamp, runtime.Format.Timestamp, runtime.UnixTimeUnit)
		value := formatValue(item.Value, runtime.Format.Value)
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
			ts := formatTimestamp(value.Timestamp, runtime.Format.Timestamp, runtime.UnixTimeUnit)
			v := formatValue(value.Value, runtime.Format.Value)
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
			ts := formatTimestamp(value.Timestamp, runtime.Format.Timestamp, runtime.UnixTimeUnit)
			v := formatValue(value.Value, runtime.Format.Value)
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

func formatTimestamp(ts model.Time, format metadata.TimestampFormat, unixTime client.UnixTimeUnit) any {
	switch format {
	case metadata.TimestampRFC3339:
		return ts.Time().Format(time.RFC3339)
	default:
		return ts.Unix() * int64(time.Second/unixTime.Duration())
	}
}

func formatValue(value model.SampleValue, format metadata.ValueFormat) any {
	switch format {
	case metadata.ValueFloat64:
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