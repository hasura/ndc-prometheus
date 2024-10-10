package metadata

import (
	"fmt"
	"regexp"
	"strings"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/iancoleman/strcase"
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

func (scb *connectorSchemaBuilder) buildNativeQueries() error {
	for name, nq := range scb.Metadata.NativeOperations.Queries {
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
	fn := schema.FunctionInfo{
		Name:        name,
		Description: query.Description,
		Arguments:   createNativeQueryArguments(),
	}
	for key, arg := range query.Arguments {
		if _, ok := fn.Arguments[key]; ok {
			return fmt.Errorf("argument `%s` is already used by the function", key)
		}
		fn.Arguments[key] = schema.ArgumentInfo{
			Description: arg.Description,
			Type:        schema.NewNamedType(string(ScalarString)).Encode(),
		}
	}

	if len(query.Labels) > 0 {
		resultType := schema.ObjectType{
			Fields: createQueryResultValuesObjectFields(),
		}

		boolExpType := schema.ObjectType{
			Description: utils.ToPtr(fmt.Sprintf("Boolean expression of the native query %s", name)),
			Fields:      schema.ObjectTypeFields{},
		}

		for key, label := range query.Labels {
			// build boolean expression argument
			boolExpType.Fields[key] = schema.ObjectField{
				Type: schema.NewNullableNamedType(objectName_NativeQueryLabelBoolExp).Encode(),
			}

			// build the result object type
			resultType.Fields[key] = schema.ObjectField{
				Description: label.Description,
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			}
		}

		objectName := fmt.Sprintf("%sResult", strcase.ToCamel(name))
		scb.ObjectTypes[objectName] = resultType

		boolExpObjectName := fmt.Sprintf("%sBoolExp", strcase.ToCamel(name))
		scb.ObjectTypes[boolExpObjectName] = boolExpType
		fn.Arguments[ArgumentKeyWhere] = schema.ArgumentInfo{
			Description: boolExpType.Description,
			Type:        schema.NewNullableNamedType(boolExpObjectName).Encode(),
		}

		fn.ResultType = schema.NewArrayType(schema.NewNamedType(objectName)).Encode()
	} else {
		fn.ResultType = schema.NewArrayType(schema.NewNamedType(objectName_QueryResultValues)).Encode()
	}

	scb.Functions[name] = fn
	return nil
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
