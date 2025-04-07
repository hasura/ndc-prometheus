package main

import (
	"testing"

	"github.com/hasura/ndc-prometheus/connector/metadata"
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
