package internal

import (
	"context"
	"fmt"
	"strconv"

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
	Request     *schema.QueryRequest
	NativeQuery *metadata.NativeQuery
	Variables   map[string]any

	arguments map[string]any
	selection schema.NestedField
}

// Explain explains the query request
func (nqe *NativeQueryExecutor) Explain(ctx context.Context) (*nativeQueryParameters, string, error) {
	span := trace.SpanFromContext(ctx)

	arguments, err := utils.ResolveArgumentVariables(nqe.Request.Arguments, nqe.Variables)
	if err != nil {
		return nil, "", schema.UnprocessableContentError("failed to resolve argument variables", map[string]any{
			"cause": err.Error(),
		})
	}
	nqe.arguments = arguments
	span.SetAttributes(utils.JSONAttribute("arguments", nqe.arguments))

	params := &nativeQueryParameters{}
	queryString := nqe.NativeQuery.Query
	for key, arg := range nqe.arguments {
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
					duration, err := ParseRangeResolution(arg)
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
	span := trace.SpanFromContext(ctx)

	arguments, err := utils.ResolveArgumentVariables(nqe.Request.Arguments, nqe.Variables)
	if err != nil {
		return nil, "", schema.UnprocessableContentError("failed to resolve argument variables", map[string]any{
			"cause": err.Error(),
		})
	}
	nqe.arguments = arguments
	span.SetAttributes(utils.JSONAttribute("arguments", nqe.arguments))

	params := &nativeQueryParameters{}
	var queryString string
	for key, arg := range nqe.arguments {
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

	var rawResults []map[string]any
	if _, ok := nqe.arguments[metadata.ArgumentKeyTime]; ok {
		rawResults, err = nqe.queryInstant(ctx, queryString, params)
	} else {
		rawResults, err = nqe.queryRange(ctx, queryString, params)
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

func (nqe *NativeQueryExecutor) queryInstant(ctx context.Context, queryString string, params *nativeQueryParameters) ([]map[string]any, error) {
	vector, _, err := nqe.Client.Query(ctx, queryString, params.Timestamp, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}
	results := createQueryResultsFromVector(vector, nqe.NativeQuery.Labels)

	return results, nil
}

func (nqe *NativeQueryExecutor) queryRange(ctx context.Context, queryString string, params *nativeQueryParameters) ([]map[string]any, error) {
	matrix, _, err := nqe.Client.QueryRange(ctx, queryString, params.Start, params.End, params.Step, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	results := createQueryResultsFromMatrix(matrix, nqe.NativeQuery.Labels)

	return results, nil
}

func createQueryResultsFromVector(vector model.Vector, labels map[string]metadata.LabelInfo) []map[string]any {
	results := make([]map[string]any, len(vector))
	for i, item := range vector {
		r := map[string]any{
			metadata.TimestampKey: item.Timestamp,
			metadata.ValueKey:     item.Value.String(),
			metadata.LabelsKey:    item.Metric,
			metadata.ValuesKey: []map[string]any{
				{
					metadata.TimestampKey: item.Timestamp,
					metadata.ValueKey:     item.Value.String(),
				},
			},
		}

		for label := range labels {
			r[label] = string(item.Metric[model.LabelName(label)])
		}

		results[i] = r
	}

	return results
}

func createQueryResultsFromMatrix(matrix model.Matrix, labels map[string]metadata.LabelInfo) []map[string]any {
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
			values[i] = map[string]any{
				metadata.TimestampKey: value.Timestamp,
				metadata.ValueKey:     value.Value.String(),
			}
			if i == valuesLen-1 {
				r[metadata.TimestampKey] = value.Timestamp
				r[metadata.ValueKey] = value.Value.String()
			}
		}

		r[metadata.ValuesKey] = values
		results[i] = r
	}

	return results
}
