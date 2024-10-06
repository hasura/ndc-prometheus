package main

import (
	"bufio"
	"bytes"
	"context"
	"fmt"
	"log/slog"
	"os"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-prometheus/connector/types"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"gopkg.in/yaml.v3"
)

var bannedLabels = []string{"__name__"}

type ExcludeLabels struct {
	Regex  *regexp.Regexp
	Labels []string
}

type updateCommand struct {
	Client        *client.Client
	OutputDir     string
	Config        *metadata.Configuration
	Include       []*regexp.Regexp
	Exclude       []*regexp.Regexp
	ExcludeLabels []ExcludeLabels
}

func introspectSchema(ctx context.Context, args *UpdateArguments) error {
	start := time.Now()
	slog.Info("introspecting metadata", slog.String("dir", args.Dir))
	originalConfig, err := metadata.ReadConfiguration(args.Dir)
	if err != nil {
		if !os.IsNotExist(err) {
			return err
		}
		originalConfig = &defaultConfiguration
	}

	apiClient, err := client.NewClient(ctx, originalConfig.ConnectionSettings)
	if err != nil {
		return err
	}

	cmd := updateCommand{
		Client:    apiClient,
		Config:    originalConfig,
		OutputDir: args.Dir,
		Include:   compileRegularExpressions(originalConfig.Generator.Metrics.Include),
		Exclude:   compileRegularExpressions(originalConfig.Generator.Metrics.Exclude),
	}

	if originalConfig.Generator.Metrics.Enabled {
		slog.Info("introspecting metrics",
			slog.String("behavior", string(originalConfig.Generator.Metrics.Behavior)),
			slog.Any("include", originalConfig.Generator.Metrics.Include),
			slog.Any("exclude", originalConfig.Generator.Metrics.Exclude),
		)
		for _, el := range originalConfig.Generator.Metrics.ExcludeLabels {
			if len(el.Labels) == 0 {
				continue
			}
			rg, err := regexp.Compile(el.Pattern)
			if err != nil {
				return fmt.Errorf("invalid exclude_labels pattern `%s`: %s", el.Pattern, err)
			}
			cmd.ExcludeLabels = append(cmd.ExcludeLabels, ExcludeLabels{
				Regex:  rg,
				Labels: el.Labels,
			})
		}
		if err := cmd.updateMetricsMetadata(ctx); err != nil {
			return err
		}
	}
	if err := cmd.validateNativeQueries(ctx); err != nil {
		return err
	}
	if err := cmd.writeConfigFile(); err != nil {
		return fmt.Errorf("failed to write the configuration file: %s", err)
	}

	slog.Info("introspected successfully", slog.String("exec_time", time.Since(start).Round(time.Millisecond).String()))
	return nil
}

func (uc *updateCommand) updateMetricsMetadata(ctx context.Context) error {
	metricsInfo, err := uc.Client.Metadata(ctx, "", "10000000")
	if err != nil {
		return err
	}

	newMetrics := map[string]metadata.MetricInfo{}
	if uc.Config.Generator.Metrics.Behavior == metadata.MetricsGenerationMerge {
		for key, metric := range uc.Config.Metadata.Metrics {
			if (len(uc.Include) > 0 && !validateRegularExpressions(uc.Include, key)) || validateRegularExpressions(uc.Exclude, key) {
				continue
			}
			newMetrics[key] = metric
		}
	}

	for key, info := range metricsInfo {
		if len(info) == 0 {
			continue
		}
		if (len(uc.Include) > 0 && !validateRegularExpressions(uc.Include, key)) ||
			validateRegularExpressions(uc.Exclude, key) ||
			len(info) == 0 {
			continue
		}
		slog.Info(key, slog.String("type", string(info[0].Type)))
		labels, err := uc.getAllLabelsOfMetric(ctx, key, info[0])
		if err != nil {
			return fmt.Errorf("error when fetching labels for metric `%s`: %s", key, err)
		}
		newMetrics[key] = metadata.MetricInfo{
			Type:        model.MetricType(info[0].Type),
			Description: &info[0].Help,
			Labels:      labels,
		}
	}
	uc.Config.Metadata.Metrics = newMetrics
	return nil
}

func (uc *updateCommand) getAllLabelsOfMetric(ctx context.Context, name string, metric v1.Metadata) (map[string]metadata.LabelInfo, error) {
	metricName := name
	if metric.Type == v1.MetricTypeHistogram || metric.Type == v1.MetricTypeGaugeHistogram {
		metricName = fmt.Sprintf("%s_count", metricName)
	}
	labels, warnings, err := uc.Client.LabelNames(ctx, []string{metricName}, uc.Config.Generator.Metrics.StartAt, time.Now(), 0)
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		slog.Debug(fmt.Sprintf("warning when fetching labels for metric `%s`", name), slog.Any("warnings", warnings))
	}
	results := make(map[string]metadata.LabelInfo)
	if len(labels) == 0 {
		return results, nil
	}
	excludedLabels := bannedLabels
	for _, el := range uc.ExcludeLabels {
		if el.Regex.MatchString(name) {
			excludedLabels = append(excludedLabels, el.Labels...)
		}
	}
	for _, key := range labels {
		if slices.Contains(excludedLabels, string(key)) {
			continue
		}

		results[string(key)] = metadata.LabelInfo{}
	}
	return results, nil
}

func (uc *updateCommand) validateNativeQueries(ctx context.Context) error {
	if len(uc.Config.Metadata.NativeOperations.Queries) == 0 {
		return nil
	}

	for key, nativeQuery := range uc.Config.Metadata.NativeOperations.Queries {
		if _, ok := uc.Config.Metadata.Metrics[key]; ok {
			return fmt.Errorf("duplicated native query name `%s`. That name may exist in the metrics collection", key)
		}
		slog.Debug(key, slog.String("type", "native_query"), slog.String("query", nativeQuery.Query))
		query := nativeQuery.Query
		for k, v := range nativeQuery.Arguments {
			switch v.Type {
			case string(metadata.ScalarInt64), string(metadata.ScalarFloat64):
				query = strings.ReplaceAll(query, fmt.Sprintf("${%s}", k), "1")
			case string(metadata.ScalarString), string(metadata.ScalarDuration), "":
				query = strings.ReplaceAll(query, fmt.Sprintf("${%s}", k), "1m")
			default:
				return fmt.Errorf("invalid argument type `%s` in the native query `%s`", k, key)
			}
		}
		_, err := uc.Client.FormatQuery(ctx, query)
		if err != nil {
			return fmt.Errorf("invalid native query %s: %s", key, err)
		}
	}

	return nil
}

func (uc *updateCommand) writeConfigFile() error {
	var buf bytes.Buffer
	writer := bufio.NewWriter(&buf)

	_, _ = writer.WriteString("# yaml-language-server: $schema=https://raw.githubusercontent.com/hasura/ndc-prometheus/main/jsonschema/configuration.json\n")
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)
	if err := encoder.Encode(uc.Config); err != nil {
		return fmt.Errorf("failed to encode the configuration file: %s", err)
	}
	writer.Flush()

	return os.WriteFile(fmt.Sprintf("%s/configuration.yaml", uc.OutputDir), buf.Bytes(), 0644)
}

var defaultConfiguration = metadata.Configuration{
	ConnectionSettings: client.ClientSettings{
		URL: types.NewEnvironmentVariable("CONNECTION_URL"),
	},
	Generator: metadata.GeneratorSettings{
		Metrics: metadata.MetricsGeneratorSettings{
			Enabled:       true,
			Behavior:      metadata.MetricsGenerationMerge,
			Include:       []string{},
			Exclude:       []string{},
			ExcludeLabels: []metadata.ExcludeLabelsSetting{},
			StartAt:       time.Date(2024, 1, 1, 0, 0, 0, 0, time.UTC),
		},
	},
	Metadata: metadata.Metadata{
		Metrics:          map[string]metadata.MetricInfo{},
		NativeOperations: metadata.NativeOperations{},
	},
	Runtime: metadata.RuntimeSettings{
		Flat:             false,
		UnixTimeUnit:     client.UnixTimeSecond,
		ConcurrencyLimit: 5,
		Format: metadata.RuntimeFormatSettings{
			Timestamp: metadata.TimestampUnix,
			Value:     metadata.ValueFloat64,
		},
	},
}

func compileRegularExpressions(inputs []string) []*regexp.Regexp {
	results := make([]*regexp.Regexp, len(inputs))
	for i, str := range inputs {
		results[i] = regexp.MustCompile(str)
	}
	return results
}

func validateRegularExpressions(patterns []*regexp.Regexp, input string) bool {
	for _, pattern := range patterns {
		if pattern.MatchString(input) {
			return true
		}
	}
	return false
}
