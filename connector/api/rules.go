package api

import (
	"context"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel/codes"
)

// FunctionPrometheusRules return a list of alerting and recording rules that are currently loaded.
// In addition it returns the currently active alerts fired by the Prometheus instance of each alerting rule.
func FunctionPrometheusRules(ctx context.Context, state *metadata.State) (v1.RulesResult, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Rules")
	defer span.End()

	results, err := state.Client.Rules(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus rules")
		span.RecordError(err)

		return v1.RulesResult{}, err
	}

	return results, nil
}
