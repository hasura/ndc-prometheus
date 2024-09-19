package metadata

import (
	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/prometheus/common/model"
	"go.opentelemetry.io/otel/trace"
)

// State the shared state of the connector
type State struct {
	Client *client.Client
	Tracer trace.Tracer
}

// Metadata the metadata configuration
type Metadata struct {
	Metrics          map[string]MetricInfo `json:"metrics" yaml:"metrics"`
	NativeOperations NativeOperations      `json:"native_operations" yaml:"native_operations"`
}

// MetricInfo the metadata information of a metric
type MetricInfo struct {
	// A metric type
	Type model.MetricType `yaml:"type" json:"type"`
	// Description of the metric
	Description *string `yaml:"description,omitempty" json:"description,omitempty"`
	// Labels returned by the metric
	Labels map[string]LabelInfo `json:"labels" yaml:"labels"`
}

// LabelInfo the information of a Prometheus label
type LabelInfo struct {
	// Description of the label
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
}
