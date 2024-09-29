package types

import (
	"fmt"
	"testing"

	"github.com/hasura/ndc-sdk-go/utils"
	"gotest.tools/v3/assert"
)

func TestEnvironmentValue(t *testing.T) {
	t.Setenv("SOME_FOO", "bar")
	testCases := []struct {
		Input    EnvironmentValue
		Expected string
		ErrorMsg string
	}{
		{
			Input:    NewEnvironmentValue("foo"),
			Expected: "foo",
		},
		{
			Input:    NewEnvironmentVariable("SOME_FOO"),
			Expected: "bar",
		},
		{
			Input:    EnvironmentValue{},
			ErrorMsg: errEnvironmentValueRequired.Error(),
		},
		{
			Input: EnvironmentValue{
				Value:    utils.ToPtr("foo"),
				Variable: utils.ToPtr("SOME_FOO"),
			},
			ErrorMsg: errEnvironmentEitherValueOrEnv.Error(),
		},
		{
			Input: EnvironmentValue{
				Variable: utils.ToPtr(""),
			},
			ErrorMsg: errEnvironmentVariableRequired.Error(),
		},
	}

	for i, tc := range testCases {
		t.Run(fmt.Sprint(i), func(t *testing.T) {
			result, err := tc.Input.Get()
			if tc.ErrorMsg != "" {
				assert.ErrorContains(t, err, tc.ErrorMsg)
			} else {
				assert.NilError(t, err)
				assert.Equal(t, result, tc.Expected)
			}
		})
	}
}
