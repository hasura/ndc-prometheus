package main

import (
	"testing"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
)

func TestNativeQueryVariables(t *testing.T) {
	testCases := []struct {
		Input             metadata.NativeQuery
		ExpectedArguments map[string]metadata.NativeQueryArgumentInfo
		ExpectedQuery     string
		ErrorMsg          string
	}{
		{
			Input: metadata.NativeQuery{
				Query: "up",
			},
			ExpectedArguments: map[string]metadata.NativeQueryArgumentInfo{},
			ExpectedQuery:     "up",
		},
		{
			Input: metadata.NativeQuery{
				Query: `up{job="${job}", instance="$instance"}`,
			},
			ExpectedArguments: map[string]metadata.NativeQueryArgumentInfo{
				"job": {
					Type: string(metadata.ScalarString),
				},
				"instance": {
					Type: string(metadata.ScalarString),
				},
			},
			ExpectedQuery: `up{job="${job}", instance="${instance}"}`,
		},
		{
			Input: metadata.NativeQuery{
				Query:     `rate(up{job="${job}", instance="$instance"}[$range])`,
				Arguments: map[string]metadata.NativeQueryArgumentInfo{},
			},
			ExpectedArguments: map[string]metadata.NativeQueryArgumentInfo{
				"job": {
					Type: string(metadata.ScalarString),
				},
				"instance": {
					Type: string(metadata.ScalarString),
				},
				"range": {
					Type: "Duration",
				},
			},
			ExpectedQuery: `rate(up{job="${job}", instance="${instance}"}[${range}])`,
		},
		{
			Input: metadata.NativeQuery{
				Query: `up{job="${job}"} > $value`,
				Arguments: map[string]metadata.NativeQueryArgumentInfo{
					"value": {
						Type: string(metadata.ScalarFloat64),
					},
				},
			},
			ExpectedArguments: map[string]metadata.NativeQueryArgumentInfo{
				"job": {
					Type: string(metadata.ScalarString),
				},
				"value": {
					Type: string(metadata.ScalarFloat64),
				},
			},
			ExpectedQuery: `up{job="${job}"} > ${value}`,
		},
		{
			Input: metadata.NativeQuery{
				Query: `up{job="${job}"} > $value`,
				Arguments: map[string]metadata.NativeQueryArgumentInfo{
					"value": {},
				},
			},
			ExpectedArguments: map[string]metadata.NativeQueryArgumentInfo{
				"job": {
					Type: string(metadata.ScalarString),
				},
				"value": {
					Type: string(metadata.ScalarString),
				},
			},
			ExpectedQuery: `up{job="${job}"} > ${value}`,
		},
		{
			Input: metadata.NativeQuery{
				Query: "up[$range",
			},
			ErrorMsg: "invalid promQL range syntax",
		},
		{
			Input: metadata.NativeQuery{
				Query: `up{job="$job}`,
			},
			ErrorMsg: "invalid promQL string syntax",
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Input.Query, func(t *testing.T) {
			uc := &updateCommand{}
			arguments, err := uc.findNativeQueryVariables(tc.Input)
			if tc.ErrorMsg != "" {
				assert.ErrorContains(t, err, tc.ErrorMsg)
				return
			}

			assert.NilError(t, err)
			assert.DeepEqual(t, arguments, tc.ExpectedArguments)
			query, err := uc.formatNativeQueryVariables(tc.Input.Query, tc.ExpectedArguments)
			assert.NilError(t, err)
			assert.Equal(t, query, tc.ExpectedQuery)
		})
	}
}

func TestInferMetricType(t *testing.T) {
	testCases := []struct {
		Name     string
		Expected model.MetricType
	}{
		{"http_requests_total", model.MetricTypeCounter},
		{"wellness_tick_session_completed_total", model.MetricTypeCounter},
		{"traces_spanmetrics_latency_bucket", model.MetricTypeHistogram},
		{"http_request_duration_seconds_bucket", model.MetricTypeHistogram},
		{"http_request_duration_seconds_sum", model.MetricTypeGauge},
		{"http_request_duration_seconds_count", model.MetricTypeGauge},
		{"target_info", model.MetricTypeGauge},
		{"process_cpu_seconds_created", model.MetricTypeGauge},
		{"go_goroutines", model.MetricTypeGauge},
		{"temperature_celsius", model.MetricTypeGauge},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := inferMetricType(tc.Name)
			assert.Equal(t, result, tc.Expected)
		})
	}
}

func TestIsHistogramSubMetric(t *testing.T) {
	testCases := []struct {
		Name     string
		NameSet  map[string]bool
		Expected bool
	}{
		{
			Name:     "http_duration_sum",
			NameSet:  map[string]bool{"http_duration_bucket": true, "http_duration_sum": true, "http_duration_count": true},
			Expected: true,
		},
		{
			Name:     "http_duration_count",
			NameSet:  map[string]bool{"http_duration_bucket": true, "http_duration_sum": true, "http_duration_count": true},
			Expected: true,
		},
		{
			Name:     "http_duration_bucket",
			NameSet:  map[string]bool{"http_duration_bucket": true, "http_duration_sum": true, "http_duration_count": true},
			Expected: false,
		},
		{
			Name:     "standalone_sum",
			NameSet:  map[string]bool{"standalone_sum": true},
			Expected: false,
		},
		{
			Name:     "standalone_count",
			NameSet:  map[string]bool{"standalone_count": true},
			Expected: false,
		},
		{
			Name:     "http_requests_total",
			NameSet:  map[string]bool{"http_requests_total": true},
			Expected: false,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			result := isHistogramSubMetric(tc.Name, tc.NameSet)
			assert.Equal(t, result, tc.Expected)
		})
	}
}
