package connector

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/hasura/ndc-prometheus/connector/internal"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"go.opentelemetry.io/otel/codes"
)

var histogramNameRegex = regexp.MustCompile(`^(\w+)_(sum|count|bucket)$`)

// Query executes a query.
func (c *PrometheusConnector) Query(ctx context.Context, configuration *metadata.Configuration, state *metadata.State, request *schema.QueryRequest) (schema.QueryResponse, error) {
	requestVars := request.Variables
	if len(requestVars) == 0 {
		requestVars = []schema.QueryRequestVariablesElem{make(schema.QueryRequestVariablesElem)}
	}
	rowSets := make([]schema.RowSet, len(requestVars))

	for i, requestVar := range requestVars {
		result, err := c.execQuery(ctx, state, request, requestVar, i)
		if err != nil {
			return nil, err
		}
		rowSets[i] = *result
	}

	return rowSets, nil
}

func (c *PrometheusConnector) execQuery(ctx context.Context, state *metadata.State, request *schema.QueryRequest, variables map[string]any, index int) (*schema.RowSet, error) {
	ctx, span := state.Tracer.Start(ctx, fmt.Sprintf("Execute Query %d", index))
	defer span.End()

	arguments, err := utils.ResolveArgumentVariables(request.Arguments, variables)
	if err != nil {
		errorMsg := "failed to resolve argument variables"
		span.SetStatus(codes.Error, errorMsg)
		span.RecordError(err)
		return nil, schema.UnprocessableContentError(errorMsg, map[string]any{
			"cause": err.Error(),
		})
	}
	span.SetAttributes(utils.JSONAttribute("arguments", arguments))

	if request.Collection == metadata.FunctionPromQLQuery {
		executor := &internal.NativeQueryExecutor{
			Tracer:      state.Tracer,
			Client:      state.Client,
			Request:     request,
			NativeQuery: &metadata.NativeQuery{},
			Arguments:   arguments,
		}
		result, err := executor.ExecuteRaw(ctx)
		if err != nil {
			span.SetStatus(codes.Error, "failed to execute raw query")
			span.RecordError(err)
			return nil, err
		}

		return result, nil
	}

	if c.apiHandler.QueryExists(request.Collection) {
		result, err := c.apiHandler.Query(ctx, state, request, arguments)
		if err != nil {
			span.SetStatus(codes.Error, "failed to execute query")
			span.RecordError(err)
			return nil, err
		}
		return result, nil
	}

	// evaluate native query
	if nativeQuery, ok := c.metadata.NativeOperations.Queries[request.Collection]; ok {
		executor := &internal.NativeQueryExecutor{
			Tracer:      state.Tracer,
			Client:      state.Client,
			Request:     request,
			NativeQuery: &nativeQuery,
			Arguments:   arguments,
		}
		result, err := executor.Execute(ctx)
		if err != nil {
			span.SetStatus(codes.Error, "failed to execute the native query")
			span.RecordError(err)
			return nil, err
		}

		return result, nil
	}

	// evaluate collection query
	histogramMatches := histogramNameRegex.FindStringSubmatch(request.Collection)

	metricNames := []string{request.Collection}
	if len(histogramMatches) > 1 {
		metricNames = []string{histogramMatches[1], request.Collection}
	}

	for _, metricName := range metricNames {
		if collection, ok := c.metadata.Metrics[metricName]; ok {
			if request.Query.Limit != nil && *request.Query.Limit <= 0 {
				return &schema.RowSet{
					Aggregates: schema.RowSetAggregates{},
					Rows:       []map[string]any{},
				}, nil
			}
			executor := &internal.QueryCollectionExecutor{
				Tracer:    state.Tracer,
				Client:    state.Client,
				Request:   request,
				Metric:    collection,
				Arguments: arguments,
				Variables: variables,
			}

			result, err := executor.Execute(ctx)
			if err != nil {
				span.SetStatus(codes.Error, "failed to execute the collection query")
				span.RecordError(err)
				return nil, err
			}

			return result, nil
		}
	}

	return nil, schema.UnprocessableContentError(fmt.Sprintf("invalid query `%s`", request.Collection), nil)
}

// QueryExplain explains a query by creating an execution plan.
func (c *PrometheusConnector) QueryExplain(ctx context.Context, conf *metadata.Configuration, state *metadata.State, request *schema.QueryRequest) (*schema.ExplainResponse, error) {
	requestVars := request.Variables
	if len(requestVars) == 0 {
		requestVars = []schema.QueryRequestVariablesElem{make(schema.QueryRequestVariablesElem)}
	}

	if slices.Contains([]string{metadata.FunctionPromQLQuery}, request.Collection) || c.apiHandler.QueryExists(request.Collection) {
		return &schema.ExplainResponse{
			Details: schema.ExplainResponseDetails{},
		}, nil
	}

	arguments, err := utils.ResolveArgumentVariables(request.Arguments, requestVars[0])
	if err != nil {
		return nil, err
	}
	if nativeQuery, ok := c.metadata.NativeOperations.Queries[request.Collection]; ok {
		executor := &internal.NativeQueryExecutor{
			Tracer:      state.Tracer,
			Client:      state.Client,
			Request:     request,
			NativeQuery: &nativeQuery,
			Arguments:   arguments,
		}
		_, queryString, err := executor.Explain(ctx)
		if err != nil {
			return nil, err
		}

		return &schema.ExplainResponse{
			Details: schema.ExplainResponseDetails{
				"query": queryString,
			},
		}, nil
	}

	// evaluate collection query
	histogramMatches := histogramNameRegex.FindStringSubmatch(request.Collection)

	metricNames := []string{request.Collection}
	if len(histogramMatches) > 1 {
		metricNames = []string{histogramMatches[1], request.Collection}
	}

	for _, metricName := range metricNames {
		if collection, ok := c.metadata.Metrics[metricName]; ok {
			executor := &internal.QueryCollectionExecutor{
				Tracer:    state.Tracer,
				Client:    state.Client,
				Request:   request,
				Metric:    collection,
				Variables: requestVars[0],
				Arguments: arguments,
			}

			_, queryString, err := executor.Explain(ctx)
			if err != nil {
				return nil, err
			}

			return &schema.ExplainResponse{
				Details: schema.ExplainResponseDetails{
					"query": queryString,
				},
			}, nil
		}
	}

	return nil, fmt.Errorf("%s: unsupported query to explain", request.Collection)
}
