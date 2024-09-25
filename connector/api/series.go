package api

import (
	"context"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel/attribute"
	"go.opentelemetry.io/otel/codes"
	"go.opentelemetry.io/otel/trace"
)

// PrometheusSeriesArguments common api arguments for the prometheus series and labels functions
type PrometheusSeriesArguments struct {
	// Repeated series selector argument that selects the series to return. At least one match[] argument must be provided
	Match []string `json:"match"`
	// Start timestamp
	Start *time.Time `json:"start"`
	// End timestamp
	End *time.Time `json:"end"`
	// Maximum number of returned series. Optional. 0 means disabled
	Limit *uint64 `json:"limit"`
}

// Validate validates arguments and options
func (psa PrometheusSeriesArguments) Validate(state *metadata.State, span trace.Span) (*PrometheusSeriesArguments, []v1.Option, error) {
	endTime := time.Now()
	arguments := PrometheusSeriesArguments{
		Match: psa.Match,
		End:   &endTime,
		Start: &time.Time{},
		Limit: psa.Limit,
	}
	if psa.Start != nil {
		arguments.Start = psa.Start
	}
	if psa.End != nil {
		arguments.End = psa.End
	}
	span.SetAttributes(attribute.StringSlice("matches", arguments.Match))
	span.SetAttributes(attribute.String("start", arguments.Start.String()))
	span.SetAttributes(attribute.String("end", arguments.End.String()))

	if len(arguments.Match) == 0 {
		errorMsg := "At least one match[] argument must be provided"
		span.SetStatus(codes.Error, errorMsg)
		return nil, nil, schema.UnprocessableContentError(errorMsg, nil)
	}

	opts, err := state.Client.ApplyOptions(span, nil)
	if err != nil {
		return nil, nil, err
	}
	if psa.Limit != nil {
		span.SetAttributes(attribute.Int64("limit", int64(*psa.Limit)))
		opts = append(opts, v1.WithLimit(*psa.Limit))
	}

	return &arguments, opts, nil
}

// FunctionPrometheusSeries find series by label matchers
func FunctionPrometheusSeries(ctx context.Context, state *metadata.State, arguments *PrometheusSeriesArguments) ([]map[string]any, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Series")
	defer span.End()

	args, _, err := arguments.Validate(state, span)
	if err != nil {
		return nil, err
	}
	var limit uint64
	if args.Limit != nil {
		limit = *args.Limit
	}
	labelSets, warnings, err := state.Client.Series(ctx, args.Match, *args.Start, *args.End, limit)
	if len(warnings) > 0 {
		span.SetAttributes(attribute.StringSlice("warnings", warnings))
	}
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus series")
		span.RecordError(err)
		return nil, err
	}

	results := make([]map[string]any, len(labelSets))
	for i, labelSet := range labelSets {
		result := make(map[string]any)
		for key, value := range labelSet {
			result[string(key)] = value
		}
		results[i] = result
	}
	return results, nil
}
