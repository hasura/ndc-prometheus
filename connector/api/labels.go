package api

import (
	"context"
	"errors"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
)

// FunctionPrometheusLabelNames return a list of label names
func FunctionPrometheusLabelNames(ctx context.Context, state *metadata.State, arguments *PrometheusSeriesArguments) ([]string, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Label Names")
	defer span.End()

	args, _, err := arguments.Validate(state, span)
	if err != nil {
		return nil, err
	}
	var limit uint64
	if args.Limit != nil {
		limit = *args.Limit
	}
	results, warnings, err := state.Client.LabelNames(ctx, args.Match, *args.Start, *args.End, limit)
	if len(warnings) > 0 {
		span.SetAttributes(attribute.StringSlice("warnings", warnings))
	}
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus labels")
		span.RecordError(err)
		return nil, err
	}

	return results, nil
}

// PrometheusLabelValuesArguments common api arguments to getting the prometheus label values
type PrometheusLabelValuesArguments struct {
	PrometheusSeriesArguments

	LabelName string `json:"label_name"`
}

// FunctionPrometheusLabelValues return a list of label values for a provided label name
func FunctionPrometheusLabelValues(ctx context.Context, state *metadata.State, arguments *PrometheusLabelValuesArguments) ([]string, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Label Values")
	defer span.End()

	if arguments.LabelName == "" {
		return nil, errors.New("label_name is required")
	}
	args, opts, err := arguments.Validate(state, span)
	if err != nil {
		return nil, err
	}
	rawResults, warnings, err := state.Client.LabelValues(ctx, arguments.LabelName, args.Match, *args.Start, *args.End, opts...)
	if len(warnings) > 0 {
		span.SetAttributes(attribute.StringSlice("warnings", warnings))
	}
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus labels")
		span.RecordError(err)
		return nil, err
	}

	results := make([]string, len(rawResults))
	for i, item := range rawResults {
		results[i] = string(item)
	}

	return results, nil
}
