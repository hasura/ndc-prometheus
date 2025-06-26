package metadata

import (
	"os"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-sdk-go/utils"
	"gopkg.in/yaml.v3"
)

// Configuration the configuration of Prometheus connector.
type Configuration struct {
	// Connection settings to connect the Prometheus server
	ConnectionSettings client.ClientSettings `json:"connection_settings" yaml:"connection_settings"`
	// Settings to generate metrics metadata
	Generator GeneratorSettings `json:"generator"           yaml:"generator"`
	// The metadata of metrics and native queries
	Metadata Metadata `json:"metadata"            yaml:"metadata"`
	// Runtime settings
	Runtime RuntimeSettings `json:"runtime"             yaml:"runtime"`
}

// MetricsGenerationBehavior the behavior of metrics generation.
type MetricsGenerationBehavior string

const (
	MetricsGenerationMerge   = "merge"
	MetricsGenerationReplace = "replace"
)

// MetricsGeneratorSettings contain settings for the metrics generation.
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

// ExcludeLabelsSetting the setting to exclude labels.
type ExcludeLabelsSetting struct {
	// The regular expression pattern of metric names
	Pattern string `json:"pattern" yaml:"pattern"`
	// List of labels to be excluded
	Labels []string `json:"labels"  yaml:"labels"`
}

// GeneratorSettings contain settings for the configuration generator.
type GeneratorSettings struct {
	Metrics MetricsGeneratorSettings `json:"metrics" yaml:"metrics"`
}

// TimestampFormat the format for timestamp serialization.
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

// ValueFormat the format for value serialization.
type ValueFormat string

const (
	ValueString  ValueFormat = "string"
	ValueFloat64 ValueFormat = "float64"
)

// RuntimeFormatSettings format settings for timestamps and values in runtime.
type RuntimeFormatSettings struct {
	// The serialization format for timestamp
	Timestamp TimestampFormat `json:"timestamp" jsonschema:"enum=rfc3339,enum=unix,enum=unix_ms,enum=unix_us,enum=unix_ns,default=unix" yaml:"timestamp"`
	// The serialization format for value
	Value ValueFormat `json:"value" jsonschema:"enum=string,enum=float64,default=string"                                    yaml:"value"`
	// The serialization format for not-a-number values
	NaN any `json:"nan" jsonschema:"oneof_type=string;number;null" yaml:"nan"`
	// The serialization format for infinite values
	Inf any `json:"inf" jsonschema:"oneof_type=string;number;null" yaml:"inf"`
	// The serialization format for negative infinite values
	NegativeInf any `json:"negative_inf" jsonschema:"oneof_type=string;number;null"                                              yaml:"negative_inf"`
}

// RuntimeSettings contain settings for the runtime engine.
type RuntimeSettings struct {
	// Enable PromptQL-compatible mode.
	PromptQL bool `json:"promptql" yaml:"promptql"`
	// Disable native Prometheus APIs.
	DisablePrometheusAPI bool `json:"disable_prometheus_api,omitempty" yaml:"disable_prometheus_api,omitempty"`
	// Flatten value points to the root array.
	// If the PromptQL mode is on the result is always flat.
	Flat bool `json:"flat" yaml:"flat"`
	// The default unit for unix timestamp.
	UnixTimeUnit UnixTimeUnit `json:"unix_time_unit" yaml:"unix_time_unit" jsonschema:"enum=s,enum=ms,enum=us,enum=ns,default=s"`
	// The serialization format for response fields.
	Format RuntimeFormatSettings `json:"format" yaml:"format"`
	// The concurrency limit of queries if there are many variables in a single query.
	ConcurrencyLimit int `json:"concurrency_limit,omitempty" yaml:"concurrency_limit,omitempty"`
}

// IsFlat gets the flat setting.
func (rs RuntimeSettings) IsFlat(flat *bool) bool {
	if rs.PromptQL {
		return true
	}

	if flat != nil {
		return *flat
	}

	return rs.Flat
}

// GetUnixTimeUnit gets the unix time unit setting.
func (rs RuntimeSettings) GetUnixTimeUnit() UnixTimeUnit {
	if rs.UnixTimeUnit == "" {
		return UnixTimeSecond
	}

	return rs.UnixTimeUnit
}

// ParseTimestamp parses timestamp from an unknown value.
func (rs RuntimeSettings) ParseTimestamp(s any) (*time.Time, error) {
	return utils.DecodeNullableDateTime(s, utils.WithBaseUnix(rs.GetUnixTimeUnit().Duration()))
}

// ParseDuration parses duration from an unknown value.
func (rs RuntimeSettings) ParseDuration(value any) (time.Duration, error) {
	return ParseDuration(value, rs.GetUnixTimeUnit())
}

// ParseRangeResolution parses the range resolution from a string.
func (rs RuntimeSettings) ParseRangeResolution(value any) (*RangeResolution, error) {
	return ParseRangeResolution(value, rs.GetUnixTimeUnit())
}

// ReadConfiguration reads the configuration from file.
func ReadConfiguration(configurationDir string) (*Configuration, error) {
	var config Configuration

	yamlBytes, err := os.ReadFile(configurationDir + "/configuration.yaml")
	if err != nil {
		return nil, err
	}

	if err = yaml.Unmarshal(yamlBytes, &config); err != nil {
		return nil, err
	}

	return &config, nil
}
