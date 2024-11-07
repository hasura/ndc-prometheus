package metadata

import (
	"fmt"
	"os"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"gopkg.in/yaml.v3"
)

// Configuration the configuration of Prometheus connector
type Configuration struct {
	// Connection settings to connect the Prometheus server
	ConnectionSettings client.ClientSettings `json:"connection_settings" yaml:"connection_settings"`
	// Settings to generate metrics metadata
	Generator GeneratorSettings `json:"generator" yaml:"generator"`
	// The metadata of metrics and native queries
	Metadata Metadata `json:"metadata" yaml:"metadata"`
	// Runtime settings
	Runtime RuntimeSettings `json:"runtime" yaml:"runtime"`
}

// MetricsGenerationBehavior the behavior of metrics generation
type MetricsGenerationBehavior string

const (
	MetricsGenerationMerge   = "merge"
	MetricsGenerationReplace = "replace"
)

// MetricsGeneratorSettings contain settings for the metrics generation
type MetricsGeneratorSettings struct {
	// Enable the metrics generation
	Enabled  bool                      `json:"enabled" yaml:"enabled"`
	Behavior MetricsGenerationBehavior `json:"behavior" yaml:"behavior" jsonschema:"enum=merge,enum=replace"`
	// Include metrics with regular expression matching. Include all metrics by default
	Include []string `json:"include" yaml:"include"`
	// Exclude metrics with regular expression matching.
	// Note: exclude is higher priority than include
	Exclude []string `json:"exclude" yaml:"exclude"`
	// Exclude unnecessary labels
	ExcludeLabels []ExcludeLabelsSetting `json:"exclude_labels" yaml:"exclude_labels"`
	// The minimum timestamp that the plugin uses to query metadata
	StartAt time.Time `json:"start_at" yaml:"start_at"`
}

// ExcludeLabelsSetting the setting to exclude labels
type ExcludeLabelsSetting struct {
	// The regular expression pattern of metric names
	Pattern string `json:"pattern" yaml:"pattern"`
	// List of labels to be excluded
	Labels []string `json:"labels" yaml:"labels"`
}

// GeneratorSettings contain settings for the configuration generator
type GeneratorSettings struct {
	Metrics MetricsGeneratorSettings `json:"metrics" yaml:"metrics"`
}

// TimestampFormat the format for timestamp serialization
type TimestampFormat string

const (
	// Represents the timestamp as a Unix timestamp in RFC3339 string.
	TimestampRFC3339 TimestampFormat = "rfc3339"
	// Represents the timestamp as a Unix timestamp in seconds.
	TimestampUnix TimestampFormat = "unix"
	// Represents the timestamp as a Unix timestamp in milliseconds.
	TimestampUnixMilli TimestampFormat = "unix_ms"
	// Represents the timestamp as a Unix timestamp in microseconds.
	TimestampUnixMicro TimestampFormat = "unix_us"
	// Represents the timestamp as a Unix timestamp in nanoseconds.
	TimestampUnixNano TimestampFormat = "unix_ns"
)

// ValueFormat the format for value serialization
type ValueFormat string

const (
	ValueString  ValueFormat = "string"
	ValueFloat64 ValueFormat = "float64"
)

// RuntimeFormatSettings format settings for timestamps and values in runtime
type RuntimeFormatSettings struct {
	// The serialization format for timestamp
	Timestamp TimestampFormat `json:"timestamp" yaml:"timestamp" jsonschema:"enum=rfc3339,enum=unix,enum=unix_ms,enum=unix_us,enum=unix_ns,default=unix"`
	// The serialization format for value
	Value ValueFormat `json:"value" yaml:"value" jsonschema:"enum=string,enum=float64,default=string"`
	// The serialization format for not-a-number values
	NaN any `json:"nan" yaml:"nan" jsonschema:"oneof_type=string;number;null"`
	// The serialization format for infinite values
	Inf any `json:"inf" yaml:"inf" jsonschema:"oneof_type=string;number;null"`
	// The serialization format for negative infinite values
	NegativeInf any `json:"negative_inf" yaml:"negative_inf" jsonschema:"oneof_type=string;number;null"`
}

// RuntimeSettings contain settings for the runtime engine
type RuntimeSettings struct {
	// Flatten value points to the root array
	Flat bool `json:"flat" yaml:"flat"`
	// The default unit for unix timestamp
	UnixTimeUnit client.UnixTimeUnit `json:"unix_time_unit" yaml:"unix_time_unit" jsonschema:"enum=s,enum=ms,enum=us,enum=ns,default=s"`
	// The serialization format for response fields
	Format RuntimeFormatSettings `json:"format" yaml:"format"`
	// The concurrency limit of queries if there are many variables in a single query
	ConcurrencyLimit int `json:"concurrency_limit,omitempty" yaml:"concurrency_limit,omitempty"`
}

// ReadConfiguration reads the configuration from file
func ReadConfiguration(configurationDir string) (*Configuration, error) {
	var config Configuration
	yamlBytes, err := os.ReadFile(fmt.Sprintf("%s/configuration.yaml", configurationDir))
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(yamlBytes, &config); err != nil {
		return nil, err
	}
	return &config, nil
}
