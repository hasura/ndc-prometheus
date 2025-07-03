package metadata

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/iancoleman/strcase"
)

// The variable syntax for native queries is ${<name>} which is compatible with Grafana.
var (
	promQLVariableRegex       = regexp.MustCompile(`\${(\w+)}`)
	allowedNativeQueryScalars = []ScalarName{
		ScalarString,
		ScalarDuration,
		ScalarInt64,
		ScalarFloat64,
	}
)

// NativeOperations the list of native query and mutation definitions.
type NativeOperations struct {
	// The definition map of native queries
	Queries map[string]NativeQuery `json:"queries" yaml:"queries"`
}

// NativeQueryArgumentInfo the input argument.
type NativeQueryArgumentInfo struct {
	// Description of the argument
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	Type        string  `json:"type"                  yaml:"type"                  jsonschema:"enum=Int64,enum=Float64,enum=String,enum=Duration"`
}

// NativeQuery contains the information a native query.
type NativeQuery struct {
	// The PromQL query string to use for the Native Query.
	// We can interpolate values using `${<varname>}` syntax,
	// such as http_requests_total{job=~"${<varname>}"}
	Query string `json:"query"                 yaml:"query"`
	// Description of the query
	Description *string `json:"description,omitempty" yaml:"description,omitempty"`
	// Labels returned by the native query
	Labels map[string]LabelInfo `json:"labels"                yaml:"labels"`
	// Information of input arguments
	Arguments map[string]NativeQueryArgumentInfo `json:"arguments"             yaml:"arguments"`
}

func (scb *connectorSchemaBuilder) buildNativeQueries() error {
	for name, nq := range scb.Configuration.Metadata.NativeOperations.Queries {
		if err := scb.checkDuplicatedOperation(name); err != nil {
			return err
		}

		if err := scb.buildNativeQuery(name, &nq); err != nil {
			return err
		}
	}

	return nil
}

func (scb *connectorSchemaBuilder) buildNativeQuery(name string, query *NativeQuery) error {
	arguments := createCollectionArguments(scb.Configuration.Runtime.PromptQL)

	for key, arg := range query.Arguments {
		if _, ok := arguments[key]; ok {
			return fmt.Errorf("argument `%s` is already used by the function", key)
		}

		scalarName := arg.Type

		if arg.Type != "" {
			if !slices.Contains(allowedNativeQueryScalars, ScalarName(arg.Type)) {
				return fmt.Errorf(
					"%s: unsupported native query argument type %s; argument: %s ",
					name,
					scalarName,
					key,
				)
			}
		} else {
			scalarName = string(ScalarString)
		}

		arguments[key] = schema.ArgumentInfo{
			Description: arg.Description,
			Type:        schema.NewNamedType(scalarName).Encode(),
		}
	}

	resultType := createMetricObjectType(scb.Configuration.Runtime.PromptQL)

	for key, label := range query.Labels {
		resultType.Fields[key] = schema.ObjectField{
			Description: label.Description,
			Type:        schema.NewNamedType(string(ScalarString)).Encode(),
		}
	}

	objectName := strcase.ToCamel(name)

	if _, ok := scb.ObjectTypes[objectName]; ok {
		objectName = strcase.ToCamel(name) + "Result"
	}

	scb.ObjectTypes[objectName] = resultType
	collection := schema.CollectionInfo{
		Name:                  name,
		Type:                  objectName,
		Arguments:             arguments,
		Description:           query.Description,
		UniquenessConstraints: schema.CollectionInfoUniquenessConstraints{},
	}

	scb.Collections[name] = collection

	return nil
}

// FindNativeQueryVariableNames find possible variables in the native query.
func FindNativeQueryVariableNames(query string) []string {
	matches := promQLVariableRegex.FindAllStringSubmatch(query, -1)
	results := make([]string, len(matches))

	for i, m := range matches {
		results[i] = m[1]
	}

	return results
}

// ReplaceNativeQueryVariable replaces the native query with variable.
func ReplaceNativeQueryVariable(query string, name string, value string) string {
	return strings.ReplaceAll(query, fmt.Sprintf("${%s}", name), value)
}

func createPromQLQueryArguments() schema.FunctionInfoArguments {
	arguments := schema.FunctionInfoArguments{}

	for _, key := range []string{ArgumentKeyStart, ArgumentKeyEnd, ArgumentKeyStep, ArgumentKeyTime, ArgumentKeyTimeout, ArgumentKeyFlat} {
		arguments[key] = defaultArgumentInfos[key]
	}

	arguments[ArgumentKeyQuery] = schema.ArgumentInfo{
		Description: utils.ToPtr("The raw promQL query"),
		Type:        schema.NewNamedType(string(ScalarString)).Encode(),
	}

	return arguments
}
