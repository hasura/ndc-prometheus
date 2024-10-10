package internal

import (
	"context"
	"fmt"
	"strconv"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/trace"
)

// nativeQueryParameters the structured arguments which is evaluated from the raw expression
type nativeQueryParameters struct {
	Timestamp any
	Start     any
	End       any
	Timeout   any
	Step      any
}

type NativeQueryExecutor struct {
	Client      *client.Client
	Tracer      trace.Tracer
	Runtime     *metadata.RuntimeSettings
	Request     *schema.QueryRequest
	NativeQuery *metadata.NativeQuery
	Arguments   map[string]any

	selection schema.NestedField
}

// Explain explains the query request
func (nqe *NativeQueryExecutor) Explain(ctx context.Context) (*nativeQueryParameters, string, error) {
	var err error
	params := &nativeQueryParameters{}
	queryString := nqe.NativeQuery.Query
	for key, arg := range nqe.Arguments {
		switch key {
		case metadata.ArgumentKeyStart:
			params.Start = arg
		case metadata.ArgumentKeyEnd:
			params.End = arg
		case metadata.ArgumentKeyStep:
			params.Step = arg
		case metadata.ArgumentKeyTime:
			params.Timestamp = arg
		case metadata.ArgumentKeyTimeout:
			params.Timeout = arg
		default:
			argInfo, ok := nqe.NativeQuery.Arguments[key]
			if ok {
				var argString string
				switch metadata.ScalarName(argInfo.Type) {
				case metadata.ScalarInt64:
					argInt, err := utils.DecodeInt[int64](arg)
					if err != nil {
						return nil, "", schema.UnprocessableContentError(err.Error(), nil)
					}
					argString = strconv.FormatInt(argInt, 10)
				case metadata.ScalarFloat64:
					argFloat, err := utils.DecodeFloat[float64](arg)
					if err != nil {
						return nil, "", schema.UnprocessableContentError(err.Error(), nil)
					}
					argString = fmt.Sprint(argFloat)
				case metadata.ScalarDuration:
					duration, err := client.ParseRangeResolution(arg, nqe.Runtime.UnixTimeUnit)
					if err != nil {
						return nil, "", schema.UnprocessableContentError(err.Error(), nil)
					}
					if duration == nil {
						return nil, "", schema.UnprocessableContentError(fmt.Sprintf("argument `%s` is required", key), nil)
					}
					argString = fmt.Sprint(duration.String())
				default:
					argString, err = utils.DecodeString(arg)
					if err != nil {
						return nil, "", schema.UnprocessableContentError(fmt.Sprintf("%s: %s", key, err.Error()), nil)
					}
				}
				queryString = metadata.ReplaceNativeQueryVariable(queryString, key, argString)
			}
		}
	}

	if unresolvedArguments := metadata.FindNativeQueryVariableNames(queryString); len(unresolvedArguments) > 0 {
		return nil, "", schema.BadRequestError(fmt.Sprintf("unresolved variables %v in the Prometheus query", unresolvedArguments), map[string]any{
			"query": queryString,
		})
	}

	return params, queryString, nil
}

// ExplainRaw explains the raw promQL query request
func (nqe *NativeQueryExecutor) ExplainRaw(ctx context.Context) (*nativeQueryParameters, string, error) {
	params := &nativeQueryParameters{}
	var err error
	var queryString string
	for key, arg := range nqe.Arguments {
		switch key {
		case metadata.ArgumentKeyStart:
			params.Start = arg
		case metadata.ArgumentKeyEnd:
			params.End = arg
		case metadata.ArgumentKeyStep:
			params.Step = arg
		case metadata.ArgumentKeyTime:
			params.Timestamp = arg
		case metadata.ArgumentKeyTimeout:
			params.Timeout = arg
		case metadata.ArgumentKeyQuery:
			queryString, err = utils.DecodeString(arg)
			if err != nil {
				return nil, "", schema.UnprocessableContentError(err.Error(), nil)
			}
			if queryString == "" {
				return nil, "", schema.UnprocessableContentError("the query argument must not be empty", nil)
			}
		}
	}

	return params, queryString, nil
}

// Execute executes the query request
func (nqe *NativeQueryExecutor) Execute(ctx context.Context) (*schema.RowSet, error) {
	ctx, span := nqe.Tracer.Start(ctx, "Execute Native Query")
	defer span.End()
	params, queryString, err := nqe.Explain(ctx)
	if err != nil {
		return nil, err
	}

	return nqe.execute(ctx, params, queryString)
}

// ExecuteRaw executes the raw promQL query request
func (nqe *NativeQueryExecutor) ExecuteRaw(ctx context.Context) (*schema.RowSet, error) {
	ctx, span := nqe.Tracer.Start(ctx, "Execute Raw PromQL Query")
	defer span.End()
	params, queryString, err := nqe.ExplainRaw(ctx)
	if err != nil {
		return nil, err
	}
	return nqe.execute(ctx, params, queryString)
}

func (nqe *NativeQueryExecutor) execute(ctx context.Context, params *nativeQueryParameters, queryString string) (*schema.RowSet, error) {
	var err error
	nqe.selection, err = utils.EvalFunctionSelectionFieldValue(nqe.Request)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	flat, err := utils.DecodeNullableBoolean(nqe.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(fmt.Sprintf("expected boolean type for the flat field, got: %v", err), map[string]any{
			"field": metadata.ArgumentKeyFlat,
		})
	}
	if flat == nil {
		flat = &nqe.Runtime.Flat
	}

	var rawResults []map[string]any
	if _, ok := nqe.Arguments[metadata.ArgumentKeyTime]; ok {
		rawResults, err = nqe.queryInstant(ctx, queryString, params, *flat)
	} else {
		rawResults, err = nqe.queryRange(ctx, queryString, params, *flat)
	}

	if err != nil {
		return nil, err
	}

	results, err := utils.EvalNestedColumnFields(nqe.selection, rawResults)
	if err != nil {
		return nil, err
	}

	return &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
		Rows: []map[string]any{
			{
				"__value": results,
			},
		},
	}, nil
}

func (nqe *NativeQueryExecutor) queryInstant(ctx context.Context, queryString string, params *nativeQueryParameters, flat bool) ([]map[string]any, error) {
	vector, _, err := nqe.Client.Query(ctx, queryString, params.Timestamp, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}
	results := createQueryResultsFromVector(vector, nqe.NativeQuery.Labels, nqe.Runtime, flat)

	return results, nil
}

func (nqe *NativeQueryExecutor) queryRange(ctx context.Context, queryString string, params *nativeQueryParameters, flat bool) ([]map[string]any, error) {
	matrix, _, err := nqe.Client.QueryRange(ctx, queryString, params.Start, params.End, params.Step, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	results := createQueryResultsFromMatrix(matrix, nqe.NativeQuery.Labels, nqe.Runtime, flat)

	return results, nil
}

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
