package metadata

import (
	"fmt"
	"slices"
	"strings"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// QueryType the enum of a query type.
type QueryType string

const (
	// InstantQuery an instant query at a single point in time.
	InstantQuery QueryType = "instant"
	// RangeQuery a query at a range of time.
	RangeQuery QueryType = "range"
)

var enumQueryTypes = []QueryType{InstantQuery, RangeQuery}

// ParseQueryType parses the QueryType from a raw string.
func ParseQueryType(input string) (QueryType, error) {
	result := QueryType(input)

	if !slices.Contains(enumQueryTypes, result) {
		return "", fmt.Errorf("invalid query type: %s", input)
	}

	return result, nil
}

// EncodeQueryName build the query name with a query type.
func EncodeQueryName(name string, queryType QueryType) string {
	return fmt.Sprintf("%s_%s", name, queryType)
}

// DecodeQueryName extracts the query name and query type from string.
func DecodeQueryName(name string) (string, QueryType, error) {
	parts := strings.Split(name, "_")
	if len(parts) < 2 {
		return "", "", fmt.Errorf("invalid query name `%s`, the query type suffix must exist", name)
	}

	queryType, err := ParseQueryType(parts[len(parts)-1])
	if err != nil {
		return "", "", fmt.Errorf("invalid query name `%s`: %w", name, err)
	}

	return strings.Join(parts[:len(parts)-1], "_"), queryType, nil
}

func createQueryResultValueObjectFields() schema.ObjectTypeFields {
	return schema.ObjectTypeFields{
		TimestampKey: schema.ObjectField{
			Description: utils.ToPtr("The timestamp when the value is calculated"),
			Type:        schema.NewNamedType(string(ScalarTimestamp)).Encode(),
		},
		ValueKey: schema.ObjectField{
			Description: utils.ToPtr("The metric value"),
			Type:        schema.NewNamedType(string(ScalarDecimal)).Encode(),
		},
	}
}

func createQueryResultValuesObjectFields() schema.ObjectTypeFields {
	return schema.ObjectTypeFields{
		LabelsKey: schema.ObjectField{
			Description: utils.ToPtr("Labels of the metric"),
			Type:        schema.NewNamedType(string(ScalarLabelSet)).Encode(),
		},
		TimestampKey: schema.ObjectField{
			Description: utils.ToPtr(
				"An instant timestamp or the last timestamp of a range query result",
			),
			Type: schema.NewNamedType(string(ScalarTimestamp)).Encode(),
		},
		ValueKey: schema.ObjectField{
			Description: utils.ToPtr(
				"Value of the instant query or the last value of a range query",
			),
			Type: schema.NewNamedType(string(ScalarDecimal)).Encode(),
		},
		ValuesKey: schema.ObjectField{
			Description: utils.ToPtr("An array of query result values"),
			Type: schema.NewArrayType(schema.NewNamedType(objectName_QueryResultValue)).
				Encode(),
		},
	}
}

func createCollectionArguments(promptql bool) schema.CollectionInfoArguments {
	arguments := schema.CollectionInfoArguments{}
	// PromptQL does not work well with arguments.
	// Arguments are temporarily disabled.
	if promptql {
		return arguments
	}

	keys := []string{ArgumentKeyStep, ArgumentKeyTimeout, ArgumentKeyOffset, ArgumentKeyFlat}

	for _, key := range keys {
		arguments[key] = defaultArgumentInfos[key]
	}

	return arguments
}
