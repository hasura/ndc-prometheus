package connector

import (
	"context"
	"fmt"
	"strings"

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

	_, explainResult, err := c.explainQueryCollection(state, request, requestVars[0], arguments)
	if err != nil {
		return nil, err
	}

	return explainResult.ToExplainResponse()
}

// evaluate collection query.
func (c *PrometheusConnector) explainQueryCollection(
	state *metadata.State,
	request *schema.QueryRequest,
	variables map[string]any,
	arguments map[string]any,
) (*internal.QueryCollectionExecutor, *internal.QueryCollectionExplainResult, error) {
	metricNames := []string{request.Collection}

	histogramMatches := histogramNameRegex.FindStringSubmatch(request.Collection)
	if len(histogramMatches) > 1 {
		metricNames = []string{histogramMatches[1], request.Collection}
	}

	executor := &internal.QueryCollectionExecutor{
		Tracer:    state.Tracer,
		Client:    state.Client,
		Runtime:   c.runtime,
		Request:   request,
		Arguments: arguments,
		Variables: variables,
	}

	for _, metricName := range metricNames {
		if collection, ok := c.metadata.Metrics[metricName]; ok {
			executor.MetricName = request.Collection
			executor.Metric = collection

			validatedRequest, err := internal.EvalCollectionRequest(request, arguments, variables, c.runtime)
			if err != nil {
				return nil, nil, schema.UnprocessableContentError(err.Error(), map[string]any{
					"collection": request.Collection,
				})
			}

			explainResult, err := executor.Explain(validatedRequest)
			if err != nil {
				return nil, nil, err
			}

			return executor, explainResult, nil
		}
	}

	// execute counter vector functions if matching.
	for _, rangeFn := range metadata.CounterRangeVectorFunctions {
		term := "_" + string(rangeFn)

		if !strings.HasSuffix(request.Collection, term) {
			continue
		}

		metricName := strings.TrimSuffix(request.Collection, term)

		collection, ok := c.metadata.Metrics[metricName]
		if !ok {
			break
		}

		executor.Metric = collection
		executor.MetricName = metricName

		validatedRequest, err := internal.EvalCollectionRequest(request, arguments, variables, c.runtime)
		if err != nil {
			return nil, nil, schema.UnprocessableContentError(err.Error(), map[string]any{
				"collection": request.Collection,
			})
		}

		executor.Functions = []internal.KeyValue{
			{
				Key:   string(rangeFn),
				Value: validatedRequest.GetStep(),
			},
		}

		explainResult, err := executor.Explain(validatedRequest)
		if err != nil {
			return nil, nil, err
		}

		return executor, explainResult, nil
	}

	// try to evaluate the quantile metric
	quantileSuffix := "_" + string(metadata.Quantile)
	if strings.HasSuffix(request.Collection, quantileSuffix) {
		metricName := strings.TrimSuffix(request.Collection, quantileSuffix)
		bucketMetricName := metricName + "_bucket"

		collection, ok := c.metadata.Metrics[metricName]
		if ok {
			executor.Metric = collection
			executor.MetricName = bucketMetricName

			validatedRequest, err := internal.EvalCollectionRequest(request, arguments, variables, c.runtime)
			if err != nil {
				return nil, nil, schema.UnprocessableContentError(err.Error(), map[string]any{
					"collection": request.Collection,
				})
			}

			// explain the histogram quantile query string with grouping.
			explainResult, err := executor.ExplainHistogramQuantile(validatedRequest)
			if err != nil {
				return nil, nil, err
			}

			return executor, explainResult, nil
		}
	}

	return nil, nil, schema.UnprocessableContentError(
		fmt.Sprintf("invalid query `%s`", request.Collection),
		nil,
	)
}
