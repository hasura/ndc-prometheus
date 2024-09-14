package connector

import (
	"context"
	"fmt"
	"regexp"
	"slices"

	"github.com/hasura/ndc-prometheus/connector/internal"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
)

var histogramNameRegex = regexp.MustCompile(`^(\w+)_(sum|count|bucket)$`)

// Query executes a query.
func (c *PrometheusConnector) Query(ctx context.Context, configuration *metadata.Configuration, state *State, request *schema.QueryRequest) (schema.QueryResponse, error) {
	requestVars := request.Variables
	if len(requestVars) == 0 {
		requestVars = []schema.QueryRequestVariablesElem{make(schema.QueryRequestVariablesElem)}
	}
	rowSets := make([]schema.RowSet, len(requestVars))
	if request.Collection == metadata.FunctionPromQLQuery {
		for i, requestVar := range requestVars {
			executor := &internal.NativeQueryExecutor{
				Tracer:      state.Tracer,
				Client:      state.Client,
				Request:     request,
				Variables:   requestVar,
				NativeQuery: &metadata.NativeQuery{},
			}
			result, err := executor.ExecuteRaw(ctx)
			if err != nil {
				return nil, err
			}

			rowSets[i] = *result
		}
		return rowSets, nil
	}

	// evaluate native query
	if nativeQuery, ok := c.metadata.NativeOperations.Queries[request.Collection]; ok {
		for i, requestVar := range requestVars {
			executor := &internal.NativeQueryExecutor{
				Tracer:      state.Tracer,
				Client:      state.Client,
				Request:     request,
				NativeQuery: &nativeQuery,
				Variables:   requestVar,
			}
			result, err := executor.Execute(ctx)
			if err != nil {
				return nil, err
			}

			rowSets[i] = *result
		}
		return rowSets, nil
	}

	// evaluate collection query
	histogramMatches := histogramNameRegex.FindStringSubmatch(request.Collection)

	metricNames := []string{request.Collection}
	if len(histogramMatches) > 1 {
		metricNames = []string{histogramMatches[1], request.Collection}
	}

	for _, metricName := range metricNames {
		if collection, ok := c.metadata.Metrics[metricName]; ok {
			for i, requestVar := range requestVars {
				if request.Query.Limit != nil && *request.Query.Limit <= 0 {
					rowSets[i] = schema.RowSet{
						Aggregates: schema.RowSetAggregates{},
						Rows:       []map[string]any{},
					}
					continue
				}
				executor := &internal.QueryCollectionExecutor{
					Tracer:    state.Tracer,
					Client:    state.Client,
					Request:   request,
					Metric:    collection,
					Variables: requestVar,
				}

				result, err := executor.Execute(ctx)
				if err != nil {
					return nil, err
				}

				rowSets[i] = *result
			}
			return rowSets, nil
		}
	}

	return nil, schema.UnprocessableContentError(fmt.Sprintf("invalid query `%s`", request.Collection), nil)
}

// QueryExplain explains a query by creating an execution plan.
func (c *PrometheusConnector) QueryExplain(ctx context.Context, conf *metadata.Configuration, state *State, request *schema.QueryRequest) (*schema.ExplainResponse, error) {
	requestVars := request.Variables
	if len(requestVars) == 0 {
		requestVars = []schema.QueryRequestVariablesElem{make(schema.QueryRequestVariablesElem)}
	}

	if slices.Contains([]string{metadata.FunctionPromQLQuery}, request.Collection) {
		return &schema.ExplainResponse{
			Details: schema.ExplainResponseDetails{},
		}, nil
	}

	if nativeQuery, ok := c.metadata.NativeOperations.Queries[request.Collection]; ok {
		executor := &internal.NativeQueryExecutor{
			Tracer:      state.Tracer,
			Client:      state.Client,
			Request:     request,
			NativeQuery: &nativeQuery,
			Variables:   requestVars[0],
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
