package metadata

import (
	"fmt"
	"os"

	"github.com/hasura/ndc-prometheus/connector/client"
	"gopkg.in/yaml.v3"
)

// Configuration the configuration of Prometheus connector
type Configuration struct {
	ConnectionSettings client.ClientSettings `json:"connection_settings" yaml:"connection_settings"`
	Generator          GeneratorSettings     `json:"generator" yaml:"generator"`
	Metadata           Metadata              `json:"metadata" yaml:"metadata"`
}

// MetricsGeneratorSettings contain settings for the metrics generation
type MetricsGeneratorSettings struct {
	Enabled bool `json:"enabled" yaml:"enabled"`
	// Include metrics with regular expression matching. Include all metrics by default
	Include []string `json:"include" yaml:"include"`
	// Exclude metrics with regular expression matching.
	// Note: exclude is higher priority than include
	Exclude []string `json:"exclude" yaml:"exclude"`
	// Exclude unnecessary labels
	ExcludeLabels []ExcludeLabelsSetting `json:"exclude_labels" yaml:"exclude_labels"`
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

// RuntimeSettings contain settings for the runtime engine
type RuntimeSettings struct {
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
