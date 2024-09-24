package api

import (
	"context"
	"strconv"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel/codes"
)

// FunctionPrometheusTargets returns an overview of the current state of the Prometheus target discovery
func FunctionPrometheusTargets(ctx context.Context, state *metadata.State) (v1.TargetsResult, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Targets")
	defer span.End()

	rawResults, err := state.Client.Targets(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus targets")
		span.RecordError(err)
		return v1.TargetsResult{}, err
	}

	return rawResults, nil
}

// PrometheusTargetsMetadataArguments the request arguments for metadata targets
type PrometheusTargetsMetadataArguments struct {
	// Label selectors that match targets by their label sets. All targets are selected if left empty
	MatchTarget *string `json:"match_target"`
	// A metric name to retrieve metadata for. All metric metadata is retrieved if left empty
	Metric *string `json:"metric"`
	// Maximum number of targets to match
	Limit *uint64 `json:"limit"`
}

// FunctionPrometheusTargetsMetadata returns metadata about metrics currently scraped from targets
func FunctionPrometheusTargetsMetadata(ctx context.Context, state *metadata.State, arguments *PrometheusTargetsMetadataArguments) ([]v1.MetricMetadata, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Targets Metadata")
	defer span.End()

	var matchTarget, metric, limit string
	if arguments.MatchTarget != nil {
		matchTarget = *arguments.MatchTarget
	}
	if arguments.Metric != nil {
		metric = *arguments.Metric
	}
	if arguments.Limit != nil {
		limit = strconv.FormatUint(*arguments.Limit, 10)
	}
	rawResults, err := state.Client.TargetsMetadata(ctx, matchTarget, metric, limit)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus metadata targets")
		span.RecordError(err)
		return nil, err
	}

	return rawResults, nil
}
