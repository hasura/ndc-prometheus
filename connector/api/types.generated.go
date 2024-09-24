// Code generated by github.com/hasura/ndc-sdk-go/cmd/hasura-ndc-go, DO NOT EDIT.
package api

import (
	"context"
	"encoding/json"
	"errors"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/connector"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"go.opentelemetry.io/otel/trace"
	"log/slog"
	"slices"
)

var connector_Decoder = utils.NewDecoder()

// FromValue decodes values from map
func (j *PrometheusLabelValuesArguments) FromValue(input map[string]any) error {
	var err error
	err = connector_Decoder.DecodeObject(&j.PrometheusSeriesArguments, input)
	if err != nil {
		return err
	}
	j.LabelName, err = utils.GetString(input, "label_name")
	if err != nil {
		return err
	}
	return nil
}

// FromValue decodes values from map
func (j *PrometheusSeriesArguments) FromValue(input map[string]any) error {
	var err error
	j.End, err = utils.GetNullableDateTime(input, "end")
	if err != nil {
		return err
	}
	j.Limit, err = utils.GetNullableUint[uint64](input, "limit")
	if err != nil {
		return err
	}
	j.Match, err = utils.GetStringSlice(input, "match")
	if err != nil {
		return err
	}
	j.Start, err = utils.GetNullableDateTime(input, "start")
	if err != nil {
		return err
	}
	return nil
}

// FromValue decodes values from map
func (j *PrometheusTargetsMetadataArguments) FromValue(input map[string]any) error {
	var err error
	j.Limit, err = utils.GetNullableUint[uint64](input, "limit")
	if err != nil {
		return err
	}
	j.MatchTarget, err = utils.GetNullableString(input, "match_target")
	if err != nil {
		return err
	}
	j.Metric, err = utils.GetNullableString(input, "metric")
	if err != nil {
		return err
	}
	return nil
}

// ToMap encodes the struct to a value map
func (j Alert) ToMap() map[string]any {
	r := make(map[string]any)
	r["activeAt"] = j.ActiveAt
	r["annotations"] = j.Annotations
	r["labels"] = j.Labels
	r["state"] = j.State
	r["value"] = j.Value

	return r
}

// ToMap encodes the struct to a value map
func (j PrometheusSeriesArguments) ToMap() map[string]any {
	r := make(map[string]any)
	r["end"] = j.End
	r["limit"] = j.Limit
	r["match"] = j.Match
	r["start"] = j.Start

	return r
}

// ScalarName get the schema name of the scalar
func (j AlertState) ScalarName() string {
	return "AlertState"
}

const (
	AlertStateFiring   AlertState = "firing"
	AlertStateInactive AlertState = "inactive"
	AlertStatePending  AlertState = "pending"
)

var enumValues_AlertState = []AlertState{AlertStateFiring, AlertStateInactive, AlertStatePending}

// ParseAlertState parses a AlertState enum from string
func ParseAlertState(input string) (AlertState, error) {
	result := AlertState(input)
	if !slices.Contains(enumValues_AlertState, result) {
		return AlertState(""), errors.New("failed to parse AlertState, expect one of AlertStateFiring, AlertStateInactive, AlertStatePending")
	}

	return result, nil
}

// IsValid checks if the value is invalid
func (j AlertState) IsValid() bool {
	return slices.Contains(enumValues_AlertState, j)
}

// UnmarshalJSON implements json.Unmarshaler.
func (j *AlertState) UnmarshalJSON(b []byte) error {
	var rawValue string
	if err := json.Unmarshal(b, &rawValue); err != nil {
		return err
	}

	value, err := ParseAlertState(rawValue)
	if err != nil {
		return err
	}

	*j = value
	return nil
}

// FromValue decodes the scalar from an unknown value
func (s *AlertState) FromValue(value any) error {
	valueStr, err := utils.DecodeNullableString(value)
	if err != nil {
		return err
	}
	if valueStr == nil {
		return nil
	}
	result, err := ParseAlertState(*valueStr)
	if err != nil {
		return err
	}

	*s = result
	return nil
}

// ScalarName get the schema name of the scalar
func (j Decimal) ScalarName() string {
	return "Decimal"
}

// DataConnectorHandler implements the data connector handler
type DataConnectorHandler struct{}

// QueryExists check if the query name exists
func (dch DataConnectorHandler) QueryExists(name string) bool {
	return slices.Contains(enumValues_FunctionName, name)
}
func (dch DataConnectorHandler) Query(ctx context.Context, state *metadata.State, request *schema.QueryRequest, rawArgs map[string]any) (*schema.RowSet, error) {
	if !dch.QueryExists(request.Collection) {
		return nil, utils.ErrHandlerNotfound
	}
	queryFields, err := utils.EvalFunctionSelectionFieldValue(request)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	result, err := dch.execQuery(ctx, state, request, queryFields, rawArgs)
	if err != nil {
		return nil, err
	}

	return &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
		Rows: []map[string]any{
			{
				"__value": result,
			},
		},
	}, nil
}

func (dch DataConnectorHandler) execQuery(ctx context.Context, state *metadata.State, request *schema.QueryRequest, queryFields schema.NestedField, rawArgs map[string]any) (any, error) {
	span := trace.SpanFromContext(ctx)
	logger := connector.GetLogger(ctx)
	switch request.Collection {
	case "prometheus_alertmanagers":

		selection, err := queryFields.AsObject()
		if err != nil {
			return nil, schema.UnprocessableContentError("the selection field type must be object", map[string]any{
				"cause": err.Error(),
			})
		}
		rawResult, err := FunctionPrometheusAlertmanagers(ctx, state)
		if err != nil {
			return nil, err
		}

		connector_addSpanEvent(span, logger, "evaluate_response_selection", map[string]any{
			"raw_result": rawResult,
		})
		result, err := utils.EvalNestedColumnObject(selection, rawResult)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "prometheus_alerts":

		selection, err := queryFields.AsArray()
		if err != nil {
			return nil, schema.UnprocessableContentError("the selection field type must be array", map[string]any{
				"cause": err.Error(),
			})
		}
		rawResult, err := FunctionPrometheusAlerts(ctx, state)
		if err != nil {
			return nil, err
		}

		connector_addSpanEvent(span, logger, "evaluate_response_selection", map[string]any{
			"raw_result": rawResult,
		})
		result, err := utils.EvalNestedColumnArrayIntoSlice(selection, rawResult)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "prometheus_label_names":

		if len(queryFields) > 0 {
			return nil, schema.UnprocessableContentError("cannot evaluate selection fields for scalar", nil)
		}
		var args PrometheusSeriesArguments
		if parseErr := args.FromValue(rawArgs); parseErr != nil {
			return nil, schema.UnprocessableContentError("failed to resolve arguments", map[string]any{
				"cause": parseErr.Error(),
			})
		}

		connector_addSpanEvent(span, logger, "execute_function", map[string]any{
			"arguments": args,
		})
		return FunctionPrometheusLabelNames(ctx, state, &args)

	case "prometheus_label_values":

		if len(queryFields) > 0 {
			return nil, schema.UnprocessableContentError("cannot evaluate selection fields for scalar", nil)
		}
		var args PrometheusLabelValuesArguments
		if parseErr := args.FromValue(rawArgs); parseErr != nil {
			return nil, schema.UnprocessableContentError("failed to resolve arguments", map[string]any{
				"cause": parseErr.Error(),
			})
		}

		connector_addSpanEvent(span, logger, "execute_function", map[string]any{
			"arguments": args,
		})
		return FunctionPrometheusLabelValues(ctx, state, &args)

	case "prometheus_rules":

		selection, err := queryFields.AsObject()
		if err != nil {
			return nil, schema.UnprocessableContentError("the selection field type must be object", map[string]any{
				"cause": err.Error(),
			})
		}
		rawResult, err := FunctionPrometheusRules(ctx, state)
		if err != nil {
			return nil, err
		}

		connector_addSpanEvent(span, logger, "evaluate_response_selection", map[string]any{
			"raw_result": rawResult,
		})
		result, err := utils.EvalNestedColumnObject(selection, rawResult)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "prometheus_series":

		if len(queryFields) > 0 {
			return nil, schema.UnprocessableContentError("cannot evaluate selection fields for scalar", nil)
		}
		var args PrometheusSeriesArguments
		if parseErr := args.FromValue(rawArgs); parseErr != nil {
			return nil, schema.UnprocessableContentError("failed to resolve arguments", map[string]any{
				"cause": parseErr.Error(),
			})
		}

		connector_addSpanEvent(span, logger, "execute_function", map[string]any{
			"arguments": args,
		})
		return FunctionPrometheusSeries(ctx, state, &args)

	case "prometheus_targets":

		selection, err := queryFields.AsObject()
		if err != nil {
			return nil, schema.UnprocessableContentError("the selection field type must be object", map[string]any{
				"cause": err.Error(),
			})
		}
		rawResult, err := FunctionPrometheusTargets(ctx, state)
		if err != nil {
			return nil, err
		}

		connector_addSpanEvent(span, logger, "evaluate_response_selection", map[string]any{
			"raw_result": rawResult,
		})
		result, err := utils.EvalNestedColumnObject(selection, rawResult)
		if err != nil {
			return nil, err
		}
		return result, nil

	case "prometheus_targets_metadata":

		selection, err := queryFields.AsArray()
		if err != nil {
			return nil, schema.UnprocessableContentError("the selection field type must be array", map[string]any{
				"cause": err.Error(),
			})
		}
		var args PrometheusTargetsMetadataArguments
		if parseErr := args.FromValue(rawArgs); parseErr != nil {
			return nil, schema.UnprocessableContentError("failed to resolve arguments", map[string]any{
				"cause": parseErr.Error(),
			})
		}

		connector_addSpanEvent(span, logger, "execute_function", map[string]any{
			"arguments": args,
		})
		rawResult, err := FunctionPrometheusTargetsMetadata(ctx, state, &args)
		if err != nil {
			return nil, err
		}

		connector_addSpanEvent(span, logger, "evaluate_response_selection", map[string]any{
			"raw_result": rawResult,
		})
		result, err := utils.EvalNestedColumnArrayIntoSlice(selection, rawResult)
		if err != nil {
			return nil, err
		}
		return result, nil

	default:
		return nil, utils.ErrHandlerNotfound
	}
}

var enumValues_FunctionName = []string{"prometheus_alertmanagers", "prometheus_alerts", "prometheus_label_names", "prometheus_label_values", "prometheus_rules", "prometheus_series", "prometheus_targets", "prometheus_targets_metadata"}

func connector_addSpanEvent(span trace.Span, logger *slog.Logger, name string, data map[string]any, options ...trace.EventOption) {
	logger.Debug(name, slog.Any("data", data))
	attrs := utils.DebugJSONAttributes(data, utils.IsDebug(logger))
	span.AddEvent(name, append(options, trace.WithAttributes(attrs...))...)
}
