package connector

import (
	"context"

	"github.com/hasura/ndc-prometheus/connector/internal"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// QueryExplain explains a query by creating an execution plan.
func (c *PrometheusConnector) QueryExplain(
	ctx context.Context,
	_ *metadata.Configuration,
	state *metadata.State,
	request *schema.QueryRequest,
) (*schema.ExplainResponse, error) {
	requestVars := request.Variables
	if len(requestVars) == 0 {
		requestVars = []schema.QueryRequestVariablesElem{make(schema.QueryRequestVariablesElem)}
	}

	if c.apiHandler.QueryExists(request.Collection) {
		return &schema.ExplainResponse{
			Details: schema.ExplainResponseDetails{},
		}, nil
	}

	arguments, err := utils.ResolveArgumentVariables(request.Arguments, requestVars[0])
	if err != nil {
		return nil, err
	}

	if request.Collection == metadata.FunctionPromQLQuery {
		executor := &internal.RawQueryExecutor{
			Tracer:    state.Tracer,
			Client:    state.Client,
			Request:   request,
			Arguments: arguments,
			Runtime:   c.runtime,
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

	if nativeQuery, ok := c.metadata.NativeOperations.Queries[request.Collection]; ok {
		executor := &internal.NativeQueryExecutor{
			Tracer:      state.Tracer,
			Client:      state.Client,
			Request:     request,
			NativeQuery: &nativeQuery,
			Arguments:   arguments,
			Runtime:     c.runtime,
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

	executor, validatedRequest, err := c.evalQueryCollection(state, request, requestVars[0], arguments)
	if err != nil {
		return nil, err
	}

	result, err := executor.Explain(ctx, validatedRequest)
	if err != nil {
		return nil, err
	}

	return result.ToExplainResponse()
}
