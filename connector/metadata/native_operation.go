package metadata

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// The variable syntax for native queries is ${<name>} which is compatible with Grafana
var promQLVariableRegex = regexp.MustCompile(`\${(\w+)}`)

// NativeOperations the list of native query and mutation definitions
type NativeOperations struct {
	// The definition map of native queries
	Queries map[string]NativeQuery `json:"queries" yaml:"queries"`
}

// NativeQueryArgumentInfo the input argument
type NativeQueryArgumentInfo struct {
	// Description of the argument
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string  `json:"type" yaml:"type" jsonschema:"enum=Int64,enum=Float64,enum=String,enum=Duration"`
}

// NativeQuery contains the information a native query
type NativeQuery struct {
	// The PromQL query string to use for the Native Query.
	// We can interpolate values using `${<varname>}` syntax,
	// such as http_requests_total{job=~"${<varname>}"}
	Query string `json:"query" yaml:"query"`
	// Description of the query
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	// Labels returned by the native query
	Labels map[string]LabelInfo `json:"labels" yaml:"labels"`
	// Information of input arguments
	Arguments map[string]NativeQueryArgumentInfo `json:"arguments" yaml:"arguments"`
}

// FindNativeQueryVariableNames find possible variables in the native query
func FindNativeQueryVariableNames(query string) []string {
	matches := promQLVariableRegex.FindAllStringSubmatch(query, -1)

	var results []string
	for _, m := range matches {
		results = append(results, m[1])
	}
	return results
}

// ReplaceNativeQueryVariable replaces the native query with variable
func ReplaceNativeQueryVariable(query string, name string, value string) string {
	return strings.ReplaceAll(query, fmt.Sprintf("${%s}", name), value)
}

func createNativeQueryArguments() schema.FunctionInfoArguments {
	arguments := schema.FunctionInfoArguments{}
	for _, key := range []string{ArgumentKeyStart, ArgumentKeyEnd, ArgumentKeyStep, ArgumentKeyTime, ArgumentKeyTimeout, ArgumentKeyFlat} {
		arguments[key] = defaultArgumentInfos[key]
	}
	return arguments
}

func createPromQLQueryArguments() schema.FunctionInfoArguments {
	arguments := createNativeQueryArguments()
	arguments[ArgumentKeyQuery] = schema.ArgumentInfo{
		Description: utils.ToPtr("The raw promQL query"),
		Type:        schema.NewNamedType(string(ScalarString)).Encode(),
	}
	return arguments
}
