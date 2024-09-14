package metadata

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestFindNativeQueryVariableNames(t *testing.T) {
	testCases := []struct {
		Input    string
		Expected []string
	}{
		{
			Input:    `rate(http_requests_total{job=~"${job}"}[${_rate_interval}])`,
			Expected: []string{"job", "_rate_interval"},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Input, func(t *testing.T) {
			matches := FindNativeQueryVariableNames(tc.Input)
			assert.DeepEqual(t, tc.Expected, matches)
		})
	}
}
