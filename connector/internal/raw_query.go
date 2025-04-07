package internal

import (
	"context"
	"fmt"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"go.opentelemetry.io/otel/trace"
)

// rawQueryParameters the structured arguments which is evaluated from the raw expression.
type rawQueryParameters struct {
	Timestamp any
	Start     any
	End       any
	Timeout   any
	Step      any
}

type RawQueryExecutor struct {
	Client    *client.Client
	Tracer    trace.Tracer
	Runtime   *metadata.RuntimeSettings
	Request   *schema.QueryRequest
	Arguments map[string]any

	selection schema.NestedField
}

// Explain explains the raw promQL query request.
func (nqe *RawQueryExecutor) Explain(ctx context.Context) (*rawQueryParameters, string, error) {
	params := &rawQueryParameters{}

	var err error

	var queryString string

	for key, arg := range nqe.Arguments {
		switch key {
		case metadata.ArgumentKeyStart:
			params.Start = arg
		case metadata.ArgumentKeyEnd:
			params.End = arg
		case metadata.ArgumentKeyStep:
			params.Step = arg
		case metadata.ArgumentKeyTime:
			params.Timestamp = arg
		case metadata.ArgumentKeyTimeout:
			params.Timeout = arg
		case metadata.ArgumentKeyQuery:
			queryString, err = utils.DecodeString(arg)
			if err != nil {
				return nil, "", schema.UnprocessableContentError(err.Error(), nil)
			}

			if queryString == "" {
				return nil, "", schema.UnprocessableContentError(
					"the query argument must not be empty",
					nil,
				)
			}
		}
	}

	return params, queryString, nil
}

// Execute executes the raw promQL query request.
func (nqe *RawQueryExecutor) Execute(ctx context.Context) (*schema.RowSet, error) {
	ctx, span := nqe.Tracer.Start(ctx, "Execute Raw PromQL Query")
	defer span.End()

	params, queryString, err := nqe.Explain(ctx)
	if err != nil {
		return nil, err
	}

	return nqe.execute(ctx, params, queryString)
}

func (nqe *RawQueryExecutor) execute(
	ctx context.Context,
	params *rawQueryParameters,
	queryString string,
) (*schema.RowSet, error) {
	var err error

	nqe.selection, err = utils.EvalFunctionSelectionFieldValue(nqe.Request)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	flat, err := utils.DecodeNullableBoolean(nqe.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(
			fmt.Sprintf("expected boolean type for the flat field, got: %v", err),
			map[string]any{
				"field": metadata.ArgumentKeyFlat,
			},
		)
	}

	if flat == nil {
		flat = &nqe.Runtime.Flat
	}

	var rawResults []map[string]any

	if _, ok := nqe.Arguments[metadata.ArgumentKeyTime]; ok {
		rawResults, err = nqe.queryInstant(ctx, queryString, params, *flat)
	} else {
		rawResults, err = nqe.queryRange(ctx, queryString, params, *flat)
	}

	if err != nil {
		return nil, err
	}

	results, err := utils.EvalNestedColumnFields(nqe.selection, rawResults)
	if err != nil {
		return nil, err
	}

	return &schema.RowSet{
		Aggregates: schema.RowSetAggregates{},
		Rows: []map[string]any{
			{
				"__value": results,
			},
		},
	}, nil
}

func (nqe *RawQueryExecutor) queryInstant(
	ctx context.Context,
	queryString string,
	params *rawQueryParameters,
	flat bool,
) ([]map[string]any, error) {
	vector, _, err := nqe.Client.Query(ctx, queryString, params.Timestamp, params.Timeout)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	results := createQueryResultsFromVector(
		vector,
		map[string]metadata.LabelInfo{},
		nqe.Runtime,
		flat,
	)

	return results, nil
}

func (nqe *RawQueryExecutor) queryRange(
	ctx context.Context,
	queryString string,
	params *rawQueryParameters,
	flat bool,
) ([]map[string]any, error) {
	matrix, _, err := nqe.Client.QueryRange(
		ctx,
		queryString,
		params.Start,
		params.End,
		params.Step,
		params.Timeout,
	)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), nil)
	}

	results := createQueryResultsFromMatrix(
		matrix,
		map[string]metadata.LabelInfo{},
		nqe.Runtime,
		flat,
	)

	return results, nil
}
