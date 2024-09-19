package connector

import (
	"context"
	"encoding/json"
	"errors"
	"fmt"
	"log/slog"

	"github.com/hasura/ndc-prometheus/connector/api"
	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/connector"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// PrometheusConnector implements a data connector for Prometheus API
type PrometheusConnector struct {
	capabilities *schema.RawCapabilitiesResponse
	rawSchema    *schema.RawSchemaResponse
	metadata     *metadata.Metadata
	apiHandler   api.DataConnectorHandler
}

// NewPrometheusConnector creates a Prometheus connector instance
func NewPrometheusConnector() *PrometheusConnector {
	return &PrometheusConnector{
		apiHandler: api.DataConnectorHandler{},
	}
}

// ParseConfiguration validates the configuration files provided by the user, returning a validated 'Configuration',
// or throwing an error to prevents Connector startup.
func (c *PrometheusConnector) ParseConfiguration(ctx context.Context, configurationDir string) (*metadata.Configuration, error) {
	restCapabilities := schema.CapabilitiesResponse{
		Version: "0.1.6",
		Capabilities: schema.Capabilities{
			Query: schema.QueryCapabilities{
				Variables:    schema.LeafCapability{},
				NestedFields: schema.NestedFieldCapabilities{},
				Explain:      map[string]any{},
			},
			Mutation: schema.MutationCapabilities{},
		},
	}
	rawCapabilities, err := json.Marshal(restCapabilities)
	if err != nil {
		return nil, fmt.Errorf("failed to encode capabilities: %s", err)
	}
	c.capabilities = schema.NewRawCapabilitiesResponseUnsafe(rawCapabilities)

	config, err := metadata.ReadConfiguration(configurationDir)
	if err != nil {
		return nil, fmt.Errorf("failed to read the configuration file at %s: %s", configurationDir, err)
	}

	c.metadata = &config.Metadata

	return config, nil
}

// TryInitState initializes the connector's in-memory state.
//
// For example, any connection pools, prepared queries,
// or other managed resources would be allocated here.
//
// In addition, this function should register any
// connector-specific metrics with the metrics registry.
func (c *PrometheusConnector) TryInitState(ctx context.Context, conf *metadata.Configuration, metrics *connector.TelemetryState) (*metadata.State, error) {
	_, span := metrics.Tracer.StartInternal(ctx, "Initialize")
	defer span.End()

	promSchema, err := metadata.BuildConnectorSchema(&conf.Metadata)
	if err != nil {
		return nil, err
	}
	ndcSchema, errs := utils.MergeSchemas(*promSchema, api.GetConnectorSchema())
	for _, e := range errs {
		slog.Warn(e.Error())
	}

	rawSchema, err := json.Marshal(ndcSchema)
	if err != nil {
		return nil, fmt.Errorf("failed to encode schema to json: %s", err)
	}
	c.rawSchema = schema.NewRawSchemaResponseUnsafe(rawSchema)

	endpoint, err := conf.ConnectionSettings.URL.Get()
	if err != nil {
		return nil, fmt.Errorf("url: %s", err)
	}
	if endpoint == "" {
		return nil, errors.New("the endpoint setting is empty")
	}
	client, err := client.NewClient(endpoint, conf.ConnectionSettings.ToHTTPClientConfig(), metrics.Tracer, conf.ConnectionSettings.Timeout)
	if err != nil {
		return nil, err
	}
	return &metadata.State{
		Client: client,
		Tracer: metrics.Tracer,
	}, nil
}

// GetSchema gets the connector's schema.
func (c *PrometheusConnector) GetSchema(ctx context.Context, configuration *metadata.Configuration, _ *metadata.State) (schema.SchemaResponseMarshaler, error) {
	return c.rawSchema, nil
}

// HealthCheck checks the health of the connector.
//
// For example, this function should check that the connector
// is able to reach its data source over the network.
//
// Should throw if the check fails, else resolve.
func (c *PrometheusConnector) HealthCheck(ctx context.Context, conf *metadata.Configuration, state *metadata.State) error {
	// return state.Client.Healthy(ctx)
	return nil
}

// GetCapabilities get the connector's capabilities.
func (c *PrometheusConnector) GetCapabilities(conf *metadata.Configuration) schema.CapabilitiesResponseMarshaler {
	return c.capabilities
}

// MutationExplain explains a mutation by creating an execution plan.
func (c *PrometheusConnector) MutationExplain(ctx context.Context, conf *metadata.Configuration, state *metadata.State, request *schema.MutationRequest) (*schema.ExplainResponse, error) {
	return nil, schema.NotSupportedError("mutation explain has not been supported yet", nil)
}

// Mutation executes a mutation.
func (c *PrometheusConnector) Mutation(ctx context.Context, configuration *metadata.Configuration, state *metadata.State, request *schema.MutationRequest) (*schema.MutationResponse, error) {
	operationResults := make([]schema.MutationOperationResults, len(request.Operations))
	for _, operation := range request.Operations {
		switch operation.Type {
		default:
			return nil, schema.BadRequestError(fmt.Sprintf("invalid operation type: %s", operation.Type), nil)
		}
	}

	return &schema.MutationResponse{
		OperationResults: operationResults,
	}, nil
}
