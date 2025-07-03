package metadata

import (
	"fmt"
	"maps"
	"slices"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/iancoleman/strcase"
	"github.com/prometheus/common/model"
)

type connectorSchemaBuilder struct {
	Configuration *Configuration
	ScalarTypes   schema.SchemaResponseScalarTypes
	ObjectTypes   schema.SchemaResponseObjectTypes
	Collections   map[string]schema.CollectionInfo
	Functions     map[string]schema.FunctionInfo
}

// BuildConnectorSchema builds the schema for the data connector from metadata.
func BuildConnectorSchema(config *Configuration) (*schema.SchemaResponse, error) {
	builder := &connectorSchemaBuilder{
		Configuration: config,
		ScalarTypes:   defaultScalars,
		ObjectTypes:   defaultObjectTypes,
		Functions: map[string]schema.FunctionInfo{
			FunctionPromQLQuery: {
				Name:        FunctionPromQLQuery,
				Description: utils.ToPtr("Execute a raw promQL query"),
				Arguments:   createPromQLQueryArguments(),
				ResultType: schema.NewArrayType(schema.NewNamedType(objectName_QueryResultValues)).
					Encode(),
			},
		},
		Collections: map[string]schema.CollectionInfo{},
	}

	// modify PromptQL-compatible schema.
	if config.Runtime.PromptQL {
		durationScalar := schema.NewScalarType()
		durationScalar.Representation = schema.NewTypeRepresentationString().Encode()
		builder.ScalarTypes[string(ScalarDuration)] = *durationScalar

		decimalScalar := schema.NewScalarType()
		decimalScalar.Representation = schema.NewTypeRepresentationFloat64().Encode()
		builder.ScalarTypes[string(ScalarDecimal)] = *decimalScalar

		builder.ScalarTypes[string(ScalarString)].ComparisonOperators[In] = schema.NewComparisonOperatorIn().Encode()
	} else {
		maps.Copy(builder.ObjectTypes, defaultFunctionObjectTypes)
	}

	if err := builder.buildMetrics(); err != nil {
		return nil, err
	}

	if err := builder.buildNativeQueries(); err != nil {
		return nil, err
	}

	return builder.buildSchemaResponse(), nil
}

func (scb *connectorSchemaBuilder) buildSchemaResponse() *schema.SchemaResponse {
	functions := make([]schema.FunctionInfo, 0, len(scb.Functions))
	collections := make([]schema.CollectionInfo, 0, len(scb.Collections))

	for _, fn := range scb.Functions {
		functions = append(functions, fn)
	}

	for _, collection := range scb.Collections {
		collections = append(collections, collection)
	}

	return &schema.SchemaResponse{
		Collections: collections,
		ObjectTypes: scb.ObjectTypes,
		Procedures:  []schema.ProcedureInfo{},
		ScalarTypes: scb.ScalarTypes,
		Functions:   functions,
	}
}

func (scb *connectorSchemaBuilder) buildMetrics() error {
	for name, info := range scb.Configuration.Metadata.Metrics {
		switch info.Type {
		case model.MetricTypeHistogram, model.MetricTypeGaugeHistogram:
			if err := scb.buildHistogramMetrics(name, info); err != nil {
				return err
			}
		default:
			if err := scb.buildCounterMetrics(name, info, info.Labels); err != nil {
				return err
			}
		}
	}

	return nil
}

func (scb *connectorSchemaBuilder) buildCounterMetrics(
	name string,
	info MetricInfo,
	labels map[string]LabelInfo,
) error {
	collection, err := scb.buildMetricsItem(name, info, labels)
	if err != nil {
		return err
	}

	// add range vector collections.
	for _, functionName := range CounterRangeVectorFunctions {
		collectionName := name + "_" + string(functionName)
		scb.Collections[collectionName] = schema.CollectionInfo{
			Name:                  collectionName,
			Type:                  collection.Type,
			Arguments:             createCollectionArguments(scb.Configuration.Runtime.PromptQL),
			Description:           info.Description,
			UniquenessConstraints: schema.CollectionInfoUniquenessConstraints{},
		}
	}

	return nil
}

func (scb *connectorSchemaBuilder) buildHistogramMetrics(
	name string,
	info MetricInfo,
) error {
	var sumCollection *schema.CollectionInfo

	for _, suffix := range []string{"sum", "count", "bucket"} {
		metricName := fmt.Sprintf("%s_%s", name, suffix)
		labels := info.Labels

		if suffix == "bucket" {
			labels["le"] = LabelInfo{}
		}

		coll, err := scb.buildMetricsItem(metricName, info, labels)
		if err != nil {
			return err
		}

		if suffix == "sum" {
			sumCollection = coll
		}
	}

	// add quantile collection.
	quantileCollectionName := name + "_" + string(Quantile)
	scb.Collections[quantileCollectionName] = schema.CollectionInfo{
		Name:                  quantileCollectionName,
		Type:                  sumCollection.Type,
		Arguments:             createCollectionArguments(scb.Configuration.Runtime.PromptQL),
		Description:           info.Description,
		UniquenessConstraints: schema.CollectionInfoUniquenessConstraints{},
	}

	return nil
}

func (scb *connectorSchemaBuilder) buildMetricsItem(
	name string,
	info MetricInfo,
	labels map[string]LabelInfo,
) (*schema.CollectionInfo, error) {
	if err := scb.checkDuplicatedOperation(name); err != nil {
		return nil, err
	}

	objectType := createMetricObjectType(scb.Configuration.Runtime.PromptQL)
	labelEnums := make([]string, 0, len(labels))

	for key, label := range labels {
		labelEnums = append(labelEnums, key)
		objectType.Fields[key] = schema.ObjectField{
			Description: label.Description,
			Type:        schema.NewNamedType(string(ScalarString)).Encode(),
		}
	}

	objectName := strcase.ToCamel(name)
	scb.ObjectTypes[objectName] = objectType
	arguments := createCollectionArguments(scb.Configuration.Runtime.PromptQL)

	if !scb.Configuration.Runtime.PromptQL {
		slices.Sort(labelEnums)
		labelEnumScalarName := objectName + "Label"
		scalarType := schema.NewScalarType()
		scalarType.Representation = schema.NewTypeRepresentationEnum(labelEnums).Encode()
		scb.ScalarTypes[labelEnumScalarName] = *scalarType
		scb.ObjectTypes[buildLabelJoinObjectTypeName(objectName)] = createLabelJoinObjectType(
			labelEnumScalarName,
		)
		scb.ObjectTypes[buildLabelReplaceObjectTypeName(objectName)] = createLabelReplaceObjectType(
			labelEnumScalarName,
		)

		// build promQL functions argument
		promQLFnsObjectName := objectName + "Functions"
		promQLFnsObject := schema.NewObjectType(
			createPromQLFunctionObjectFields(objectName, labelEnumScalarName),
			schema.ObjectTypeForeignKeys{},
			nil,
		)

		for _, fnName := range []PromQLFunctionName{Sum, Min, Max, Avg, Stddev, Stdvar, Count, Group} {
			promQLFnsObject.Fields[string(fnName)] = schema.ObjectField{
				Type: schema.NewNullableType(schema.NewArrayType(schema.NewNamedType(labelEnumScalarName))).
					Encode(),
			}
		}

		promQLFnsObject.Fields[string(CountValues)] = schema.ObjectField{
			Type: schema.NewNullableType(schema.NewNamedType(labelEnumScalarName)).Encode(),
		}

		scb.ObjectTypes[promQLFnsObjectName] = promQLFnsObject

		arguments[ArgumentKeyFunctions] = schema.ArgumentInfo{
			Description: utils.ToPtr("PromQL aggregation operators and functions for " + name),
			Type: schema.NewNullableType(schema.NewArrayType(schema.NewNamedType(promQLFnsObjectName))).
				Encode(),
		}
	}

	collection := schema.CollectionInfo{
		Name:                  name,
		Type:                  objectName,
		Arguments:             arguments,
		Description:           info.Description,
		UniquenessConstraints: schema.CollectionInfoUniquenessConstraints{},
	}
	scb.Collections[name] = collection

	return &collection, nil
}

func (scb *connectorSchemaBuilder) checkDuplicatedOperation(name string) error {
	err := fmt.Errorf("duplicated operation name: %s", name)
	if _, ok := scb.Functions[name]; ok {
		return err
	}

	if _, ok := scb.Collections[name]; ok {
		return err
	}

	return nil
}
