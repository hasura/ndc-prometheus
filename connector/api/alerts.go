package api

import (
	"context"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/codes"
)

// AlertState represents an alert state enum.
// @enum firing, inactive, pending.
type AlertState string

// Alert models an active alert.
type Alert struct {
	ActiveAt    time.Time      `json:"active_at"`
	Annotations model.LabelSet `json:"annotations"`
	Labels      model.LabelSet `json:"labels"`
	State       AlertState     `json:"state"`
	Value       Decimal        `json:"value"`
}

// FunctionPrometheusRules return a list of all active alerts.
func FunctionPrometheusAlerts(ctx context.Context, state *metadata.State) ([]Alert, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Alerts")
	defer span.End()

	rawResults, err := state.Client.Alerts(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus alerts")
		span.RecordError(err)

		return nil, err
	}

	results := make([]Alert, len(rawResults.Alerts))

	for i, item := range rawResults.Alerts {
		value, err := NewDecimal(item.Value)
		if err != nil {
			return nil, schema.InternalServerError(
				"failed to decode alert value "+item.Value,
				map[string]any{
					"error": err.Error(),
				},
			)
		}

		r := Alert{
			ActiveAt:    item.ActiveAt,
			Annotations: item.Annotations,
			Labels:      item.Labels,
			State:       AlertState(item.State),
			Value:       value,
		}
		results[i] = r
	}

	return results, nil
}

// FunctionPrometheusAlertmanagers return an overview of the current state of the Prometheus alertmanager discovery.
func FunctionPrometheusAlertmanagers(
	ctx context.Context,
	state *metadata.State,
) (v1.AlertManagersResult, error) {
	ctx, span := state.Tracer.Start(ctx, "Prometheus Alertmanagers")
	defer span.End()

	results, err := state.Client.AlertManagers(ctx)
	if err != nil {
		span.SetStatus(codes.Error, "failed to get Prometheus alertmanagers")
		span.RecordError(err)

		return v1.AlertManagersResult{}, err
	}

	return results, nil
}
