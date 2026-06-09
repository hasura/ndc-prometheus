package main

import (
	"bufio"
	"bytes"
	"context"
	"errors"
	"fmt"
	"log/slog"
	"math"
	"os"
	"regexp"
	"slices"
	"strings"
	"sync"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
	"gopkg.in/yaml.v3"
)

var (
	bannedLabels             = []string{"__name__"}
	nativeQueryVariableRegex = regexp.MustCompile(`\$\{?([a-zA-Z_]\w+)\}?`)
	allowedMetricTypes       = []model.MetricType{
		model.MetricTypeCounter,
		model.MetricTypeGauge,
		model.MetricTypeHistogram,
		model.MetricTypeGaugeHistogram,
	}
)

type ExcludeLabels struct {
	Regex  *regexp.Regexp
	Labels []string
}

// UpdateArguments represent input arguments of the `update` command.
type UpdateArguments struct {
	Dir        string `default:"." env:"HASURA_PLUGIN_CONNECTOR_CONTEXT_PATH" help:"The directory where the configuration.yaml file is present" short:"d"`
	Coroutines int    `default:"5"                                            help:"The maximum number of coroutines"                           short:"c"`
}

type updateCommand struct {
	Client        *client.Client
	OutputDir     string
	Config        *metadata.Configuration
	Include       []*regexp.Regexp
	Exclude       []*regexp.Regexp
	ExcludeLabels []ExcludeLabels

	coroutines      int
	apiFormatExists bool
	existedMetrics  map[string]any
	lock            sync.Mutex
}

// SetMetadataMetric sets the metadata metric item.
func (uc *updateCommand) SetMetadataMetric(key string, info metadata.MetricInfo) {
	uc.lock.Lock()
	defer uc.lock.Unlock()

	uc.Config.Metadata.Metrics[key] = info
	uc.existedMetrics[key] = true
}

// MetricExists check if the metric exists.
func (uc *updateCommand) MetricExists(key string) bool {
	uc.lock.Lock()
	defer uc.lock.Unlock()

	_, ok := uc.existedMetrics[key]

	return ok
}

// SetMetricExists marks if the metric as existent.
func (uc *updateCommand) SetMetricExists(key string) {
	uc.lock.Lock()
	defer uc.lock.Unlock()

	uc.existedMetrics[key] = true
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
		Client:         apiClient,
		Config:         originalConfig,
		OutputDir:      args.Dir,
		Include:        compileRegularExpressions(originalConfig.Generator.Metrics.Include),
		Exclude:        compileRegularExpressions(originalConfig.Generator.Metrics.Exclude),
		coroutines:     int(math.Max(1, float64(args.Coroutines))),
		existedMetrics: make(map[string]any),
	}

	if err := cmd.introspectMetrics(ctx, originalConfig); err != nil {
		return err
	}

	if err := cmd.validateNativeQueries(ctx); err != nil {
		return err
	}

	if err := cmd.writeConfigFile(); err != nil {
		return fmt.Errorf("failed to write the configuration file: %w", err)
	}

	slog.Info(
		"introspected successfully",
		slog.String("exec_time", time.Since(start).Round(time.Millisecond).String()),
	)

	return nil
}

func (uc *updateCommand) introspectMetrics(
	ctx context.Context,
	originalConfig *metadata.Configuration,
) error {
	if !originalConfig.Generator.Metrics.Enabled {
		return nil
	}

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
			return fmt.Errorf("invalid exclude_labels pattern `%s`: %w", el.Pattern, err)
		}

		uc.ExcludeLabels = append(uc.ExcludeLabels, ExcludeLabels{
			Regex:  rg,
			Labels: el.Labels,
		})
	}

	return uc.updateMetricsMetadata(ctx)
}

func (uc *updateCommand) updateMetricsMetadata(ctx context.Context) error {
	metricsInfo, err := uc.Client.Metadata(ctx, "", "10000000")
	if err != nil {
		return err
	}

	existingMetrics := map[string]metadata.MetricInfo{}

	if uc.Config.Generator.Metrics.Behavior == metadata.MetricsGenerationMerge {
		for key, metric := range uc.Config.Metadata.Metrics {
			if !slices.Contains(allowedMetricTypes, metric.Type) {
				continue
			}

			if (len(uc.Include) > 0 && !validateRegularExpressions(uc.Include, key)) ||
				validateRegularExpressions(uc.Exclude, key) {
				continue
			}

			existingMetrics[key] = metric
		}
	}

	uc.Config.Metadata.Metrics = make(map[string]metadata.MetricInfo)

	// If the metadata API returned empty results (common with remote-write sources
	// like OpenTelemetry Collector), fall back to discovering metrics via
	// /api/v1/label/__name__/values and infer types from metric name conventions.
	if len(metricsInfo) == 0 {
		slog.Info("metadata API returned empty results, falling back to label values discovery")

		if err := uc.updateMetricsFromLabelValues(ctx); err != nil {
			return err
		}
	} else {
		var eg errgroup.Group

		eg.SetLimit(uc.coroutines)

		for key, metricInfos := range metricsInfo {
			if len(metricInfos) == 0 {
				continue
			}

			func(k string, infos []v1.Metadata) {
				eg.Go(func() error {
					return uc.introspectMetric(ctx, k, infos)
				})
			}(key, metricInfos)
		}

		if err := eg.Wait(); err != nil {
			return err
		}
	}

	// merge existing metrics
	for key, metric := range existingMetrics {
		if _, ok := uc.existedMetrics[key]; ok {
			continue
		}

		uc.Config.Metadata.Metrics[key] = metric
	}

	return nil
}

// updateMetricsFromLabelValues discovers metrics by querying /api/v1/label/__name__/values
// and infers metric types from naming conventions. This is the fallback path when
// /api/v1/metadata returns empty (e.g. metrics ingested via remote write).
func (uc *updateCommand) updateMetricsFromLabelValues(ctx context.Context) error {
	metricNames, warnings, err := uc.Client.GetLabelValues(
		ctx,
		"__name__",
		nil,
		uc.Config.Generator.Metrics.StartAt,
		time.Now(),
		0,
	)
	if err != nil {
		return fmt.Errorf("failed to fetch metric names via label values: %w", err)
	}

	if len(warnings) > 0 {
		slog.Debug("warnings when fetching __name__ label values", slog.Any("warnings", warnings))
	}

	// Filter out sub-metric names that are part of histograms (e.g. _bucket, _sum, _count)
	// so we only introspect the base metric name for histograms.
	nameSet := make(map[string]bool, len(metricNames))
	for _, name := range metricNames {
		nameSet[string(name)] = true
	}

	var eg errgroup.Group

	eg.SetLimit(uc.coroutines)

	for _, name := range metricNames {
		metricName := string(name)

		// Skip histogram sub-metrics if the base histogram exists.
		if isHistogramSubMetric(metricName, nameSet) {
			continue
		}

		func(key string) {
			eg.Go(func() error {
				return uc.introspectMetricFromName(ctx, key)
			})
		}(metricName)
	}

	return eg.Wait()
}

// introspectMetricFromName introspects a single metric discovered via label values,
// inferring its type from naming conventions and fetching its labels.
func (uc *updateCommand) introspectMetricFromName(ctx context.Context, key string) error {
	if (len(uc.Include) > 0 && !validateRegularExpressions(uc.Include, key)) ||
		validateRegularExpressions(uc.Exclude, key) {
		return nil
	}

	metricType := inferMetricType(key)
	if !slices.Contains(allowedMetricTypes, metricType) {
		return nil
	}

	if uc.MetricExists(key) {
		slog.Warn(fmt.Sprintf("metric %s exists", key))
		return nil
	}

	switch metricType {
	case model.MetricTypeGauge, model.MetricTypeGaugeHistogram:
		for _, suffix := range []string{"sum", "bucket", "count"} {
			uc.SetMetricExists(fmt.Sprintf("%s_%s", key, suffix))
		}
	case model.MetricTypeHistogram:
		for _, suffix := range []string{"sum", "bucket", "count"} {
			uc.SetMetricExists(fmt.Sprintf("%s_%s", key, suffix))
		}
	default:
	}

	slog.Info(key, slog.String("type", string(metricType)), slog.String("source", "label_values"))

	// Build a synthetic v1.Metadata to reuse getAllLabelsOfMetric
	syntheticMeta := v1.Metadata{
		Type: v1.MetricType(metricType),
		Help: "",
	}

	labels, err := uc.getAllLabelsOfMetric(ctx, key, syntheticMeta)
	if err != nil {
		return fmt.Errorf("error when fetching labels for metric `%s`: %w", key, err)
	}

	emptyDesc := ""
	uc.SetMetadataMetric(key, metadata.MetricInfo{
		Type:        metricType,
		Description: &emptyDesc,
		Labels:      labels,
	})

	return nil
}

// inferMetricType infers the Prometheus metric type from the metric name
// using standard naming conventions.
func inferMetricType(name string) model.MetricType {
	switch {
	case strings.HasSuffix(name, "_total"):
		return model.MetricTypeCounter
	case strings.HasSuffix(name, "_bucket"):
		return model.MetricTypeHistogram
	case strings.HasSuffix(name, "_sum") || strings.HasSuffix(name, "_count"):
		// These could be histogram sub-metrics or standalone counters.
		// Default to gauge since they'll be filtered as histogram sub-metrics
		// if the base histogram exists.
		return model.MetricTypeGauge
	case strings.HasSuffix(name, "_info"):
		return model.MetricTypeGauge
	case strings.HasSuffix(name, "_created"):
		return model.MetricTypeGauge
	default:
		// Default to gauge for metrics without a recognizable suffix
		return model.MetricTypeGauge
	}
}

// isHistogramSubMetric checks if a metric name is a sub-metric of a histogram
// (i.e. _bucket, _sum, or _count) where the base histogram metric also exists.
func isHistogramSubMetric(name string, nameSet map[string]bool) bool {
	for _, suffix := range []string{"_bucket", "_sum", "_count"} {
		if strings.HasSuffix(name, suffix) {
			base := strings.TrimSuffix(name, suffix)
			// If the base+_bucket exists, this is part of a histogram
			if suffix != "_bucket" && nameSet[base+"_bucket"] {
				return true
			}
			// _bucket itself is the indicator; check if _sum or _count also exist
			if suffix == "_bucket" {
				return false // _bucket is the canonical histogram metric we keep
			}
		}
	}

	return false
}

func (uc *updateCommand) introspectMetric(
	ctx context.Context,
	key string,
	infos []v1.Metadata,
) error {
	if (len(uc.Include) > 0 && !validateRegularExpressions(uc.Include, key)) ||
		validateRegularExpressions(uc.Exclude, key) {
		return nil
	}

	for _, info := range infos {
		metricType := model.MetricType(info.Type)
		if !slices.Contains(allowedMetricTypes, metricType) {
			continue
		}

		if uc.MetricExists(key) {
			slog.Warn(fmt.Sprintf("metric %s exists", key))
		}

		switch metricType {
		case model.MetricTypeGauge, model.MetricTypeGaugeHistogram:
			for _, suffix := range []string{"sum", "bucket", "count"} {
				uc.SetMetricExists(fmt.Sprintf("%s_%s", key, suffix))
			}
		default:
		}

		slog.Info(key, slog.String("type", string(info.Type)))

		labels, err := uc.getAllLabelsOfMetric(ctx, key, info)
		if err != nil {
			return fmt.Errorf("error when fetching labels for metric `%s`: %w", key, err)
		}

		uc.SetMetadataMetric(key, metadata.MetricInfo{
			Type:        model.MetricType(info.Type),
			Description: &info.Help,
			Labels:      labels,
		})

		break
	}

	return nil
}

func (uc *updateCommand) getAllLabelsOfMetric(
	ctx context.Context,
	name string,
	metric v1.Metadata,
) (map[string]metadata.LabelInfo, error) {
	metricName := name

	if metric.Type == v1.MetricTypeHistogram || metric.Type == v1.MetricTypeGaugeHistogram {
		metricName += "_count"
	}

	labels, warnings, err := uc.Client.LabelNames(
		ctx,
		[]string{metricName},
		uc.Config.Generator.Metrics.StartAt,
		time.Now(),
		0,
	)
	if err != nil {
		return nil, err
	}

	if len(warnings) > 0 {
		slog.Debug(
			fmt.Sprintf("warning when fetching labels for metric `%s`", name),
			slog.Any("warnings", warnings),
		)
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
		if slices.Contains(excludedLabels, key) {
			continue
		}

		results[key] = metadata.LabelInfo{}
	}

	return results, nil
}

func (uc *updateCommand) validateNativeQueries(ctx context.Context) error {
	if len(uc.Config.Metadata.NativeOperations.Queries) == 0 {
		return nil
	}

	uc.checkAPIFormatQueryExist(ctx)

	newNativeQueries := make(map[string]metadata.NativeQuery)

	for key, nativeQuery := range uc.Config.Metadata.NativeOperations.Queries {
		if _, ok := uc.Config.Metadata.Metrics[key]; ok {
			return fmt.Errorf(
				"duplicated native query name `%s`. That name may exist in the metrics collection",
				key,
			)
		}

		slog.Debug(
			key,
			slog.String("type", "native_query"),
			slog.String("query", nativeQuery.Query),
		)

		args, err := uc.findNativeQueryVariables(nativeQuery)
		if err != nil {
			return fmt.Errorf("%w; query: %s", err, nativeQuery.Query)
		}

		nativeQuery.Arguments = args
		query := nativeQuery.Query

		// validate arguments and promQL syntaxes
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

		err = uc.validateQuery(ctx, query)
		if err != nil {
			return fmt.Errorf("invalid native query %s: %w", key, err)
		}

		// format and replace $<name> to ${<name>}
		query, err = uc.formatNativeQueryVariables(nativeQuery.Query, nativeQuery.Arguments)
		if err != nil {
			return err
		}

		nativeQuery.Query = query
		newNativeQueries[key] = nativeQuery
	}

	uc.Config.Metadata.NativeOperations.Queries = newNativeQueries

	return nil
}

func (uc *updateCommand) checkAPIFormatQueryExist(ctx context.Context) {
	_, err := uc.Client.FormatQuery(ctx, "up")

	uc.apiFormatExists = err == nil
	if err != nil {
		slog.Debug(
			"failed to request /api/v1/format_query endpoint",
			slog.String("error", err.Error()),
		)
	}
}

func (uc *updateCommand) validateQuery(ctx context.Context, query string) error {
	if uc.apiFormatExists {
		_, err := uc.Client.FormatQuery(ctx, query)

		return err
	}

	now := time.Now()
	_, _, err := uc.Client.Query(ctx, query, &now, 30*time.Second)

	return err
}

func (uc *updateCommand) writeConfigFile() error {
	var buf bytes.Buffer

	writer := bufio.NewWriter(&buf)

	_, _ = writer.WriteString(
		"# yaml-language-server: $schema=https://raw.githubusercontent.com/hasura/ndc-prometheus/main/jsonschema/configuration.json\n",
	)
	encoder := yaml.NewEncoder(writer)
	encoder.SetIndent(2)

	if err := encoder.Encode(uc.Config); err != nil {
		return fmt.Errorf("failed to encode the configuration file: %w", err)
	}

	_ = writer.Flush()

	return os.WriteFile(uc.OutputDir+"/configuration.yaml", buf.Bytes(), 0o644)
}

func (uc *updateCommand) findNativeQueryVariables(
	nq metadata.NativeQuery,
) (map[string]metadata.NativeQueryArgumentInfo, error) {
	result := map[string]metadata.NativeQueryArgumentInfo{}

	matches := nativeQueryVariableRegex.FindAllStringSubmatchIndex(nq.Query, -1)
	if len(matches) == 0 {
		return result, nil
	}

	queryLength := len(nq.Query)

	for _, match := range matches {
		if len(match) < 4 {
			continue
		}

		name, argumentInfo, err := uc.evalMatchedNativeQuery(nq, match, queryLength)
		if err != nil {
			return nil, err
		}

		result[name] = *argumentInfo
	}

	return result, nil
}

func (uc *updateCommand) evalMatchedNativeQuery(
	nq metadata.NativeQuery,
	match []int,
	queryLength int,
) (string, *metadata.NativeQueryArgumentInfo, error) {
	name := nq.Query[match[2]:match[3]]
	argumentInfo := metadata.NativeQueryArgumentInfo{}

	if match[0] > 0 && nq.Query[match[0]-1] == '[' {
		// duration variables should be bounded by square brackets
		if match[1] >= queryLength || nq.Query[match[1]] != ']' {
			return "", nil, errors.New("invalid promQL range syntax")
		}

		argumentInfo.Type = string(metadata.ScalarDuration)
	} else if match[0] > 0 && nq.Query[match[0]-1] == '"' {
		// duration variables should be bounded by double quotes
		if match[1] >= queryLength || nq.Query[match[1]] != '"' {
			return "", nil, errors.New("invalid promQL string syntax")
		}

		argumentInfo.Type = string(metadata.ScalarString)
	}

	if len(nq.Arguments) > 0 {
		// merge the existing argument from the configuration file
		arg, ok := nq.Arguments[name]
		if ok {
			argumentInfo.Description = arg.Description

			if argumentInfo.Type == "" && arg.Type != "" {
				argumentInfo.Type = arg.Type
			}
		}
	}

	if argumentInfo.Type == "" {
		argumentInfo.Type = string(metadata.ScalarString)
	}

	return name, &argumentInfo, nil
}

func (uc *updateCommand) formatNativeQueryVariables(
	queryInput string,
	variables map[string]metadata.NativeQueryArgumentInfo,
) (string, error) {
	query := queryInput

	for key := range variables {
		rawPattern := fmt.Sprintf(`\$\{?%s\}?`, key)

		rg, err := regexp.Compile(rawPattern)
		if err != nil {
			return "", fmt.Errorf(
				"failed to compile regular expression %s, query: %s, error: %w",
				rawPattern,
				queryInput,
				err,
			)
		}

		query = rg.ReplaceAllLiteralString(query, fmt.Sprintf("${%s}", key))
	}

	return query, nil
}

var defaultConfiguration = metadata.Configuration{
	ConnectionSettings: client.ClientSettings{
		URL: utils.NewEnvStringValue("CONNECTION_URL"),
	},
	Generator: metadata.GeneratorSettings{
		Metrics: metadata.MetricsGeneratorSettings{
			Enabled:       true,
			Behavior:      metadata.MetricsGenerationReplace,
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
		PromptQL:             false,
		Flat:                 false,
		DisablePrometheusAPI: false,
		UnixTimeUnit:         metadata.UnixTimeSecond,
		ConcurrencyLimit:     5,
		Format: metadata.RuntimeFormatSettings{
			Timestamp:   metadata.TimestampUnix,
			Value:       metadata.ValueFloat64,
			NaN:         "NaN",
			Inf:         "+Inf",
			NegativeInf: "-Inf",
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
