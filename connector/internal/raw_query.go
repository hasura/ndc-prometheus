package internal

import (
	"context"
	"fmt"
	"time"

	"github.com/hasura/ndc-prometheus/connector/client"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"go.opentelemetry.io/otel/trace"
)

// rawQueryParameters the structured arguments which is evaluated from the raw expression.
type rawQueryParameters struct {
	Timestamp *time.Time
	Range     *v1.Range
	Timeout   time.Duration
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
	var start, end *time.Time
	var step time.Duration

	for key, arg := range nqe.Arguments {
		switch key {
		case metadata.ArgumentKeyStart:
			start, err = nqe.Runtime.ParseTimestamp(arg)
			if err != nil {
				return nil, "", err
			}
		case metadata.ArgumentKeyEnd:
			end, err = nqe.Runtime.ParseTimestamp(arg)
			if err != nil {
				return nil, "", err
			}
		case metadata.ArgumentKeyStep:
			step, err = nqe.Runtime.ParseDuration(arg)
			if err != nil {
				return nil, "", err
			}
		case metadata.ArgumentKeyTime:
			params.Timestamp, err = nqe.Runtime.ParseTimestamp(arg)
			if err != nil {
				return nil, "", err
			}
		case metadata.ArgumentKeyTimeout:
			params.Timeout, err = nqe.Runtime.ParseDuration(arg)
			if err != nil {
				return nil, "", err
			}
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

	if start != nil || end != nil {
		params.Range, err = metadata.NewRange(start, end, step)
		if err != nil {
			return nil, "", schema.UnprocessableContentError(err.Error(), nil)
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

	nullableFlat, err := utils.DecodeNullableBoolean(nqe.Arguments[metadata.ArgumentKeyFlat])
	if err != nil {
		return nil, schema.UnprocessableContentError(
			fmt.Sprintf("expected boolean type for the flat field, got: %v", err),
			map[string]any{
				"field": metadata.ArgumentKeyFlat,
			},
		)
	}

	flat := nqe.Runtime.IsFlat(nullableFlat)
	var rawResults []map[string]any

	if params.Range != nil {
		rawResults, err = nqe.queryRange(ctx, queryString, params, flat)
	} else {
		rawResults, err = nqe.queryInstant(ctx, queryString, params, flat)
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
		*params.Range,
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
