package internal

import (
	"context"
	"sync"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/prometheus/common/model"
	"golang.org/x/sync/errgroup"
)

func (qce *QueryCollectionExecutor) queryAggregates(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
) (map[string]any, error) {
	aggregateLength := len(explainResult.Aggregates)
	if aggregateLength == 0 {
		return map[string]any{}, nil
	}

	timestamp := explainResult.Request.Timestamp
	if timestamp == nil {
		if explainResult.Request.Range != nil {
			timestamp = &explainResult.Request.Range.End
		} else {
			now := time.Now()
			timestamp = &now
		}
	}

	result := NewAggregateResults()

	if aggregateLength == 1 {
		for key, agg := range explainResult.Aggregates {
			err := qce.queryAggregate(ctx, explainResult, result, key, agg, timestamp)
			if err != nil {
				return nil, err
			}

			return result.results, nil
		}
	}

	concurrencyLimit := qce.Runtime.ConcurrencyLimit
	if concurrencyLimit <= 0 {
		concurrencyLimit = aggregateLength
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(concurrencyLimit)

	for key, aggQuery := range explainResult.Aggregates {
		eg.Go(func() error {
			return qce.queryAggregate(ctx, explainResult, result, key, aggQuery, timestamp)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return result.results, nil
}

func (qce *QueryCollectionExecutor) queryAggregate(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
	result *AggregateResults,
	key, query string,
	timestamp *time.Time,
) error {
	vector, _, err := qce.Client.Query(ctx, query, timestamp, explainResult.Request.Timeout)
	if err != nil {
		return schema.UnprocessableContentError(err.Error(), nil)
	}

	vectorLength := len(vector)

	if len(vector) == 0 {
		result.SetResult(key, nil)
	} else {
		result.SetResult(key, formatValue(vector[vectorLength-1].Value, qce.Runtime.Format))
	}

	return nil
}

func (qce *QueryCollectionExecutor) queryGroupAggregates(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
) ([]schema.Group, error) {
	if explainResult.Groups == nil || len(explainResult.Groups.AggregateQueries) == 0 {
		return nil, nil
	}

	results := NewGroupResults(qce.Runtime)
	aggregateLength := len(explainResult.Groups.AggregateQueries)

	if aggregateLength == 1 {
		for key, aggQuery := range explainResult.Groups.AggregateQueries {
			err := qce.queryGroupAggregate(ctx, explainResult, results, key, aggQuery)
			if err != nil {
				return nil, err
			}

			return results.groups, nil
		}
	}

	concurrencyLimit := qce.Runtime.ConcurrencyLimit
	if concurrencyLimit <= 0 {
		concurrencyLimit = aggregateLength
	}

	eg, ctx := errgroup.WithContext(ctx)
	eg.SetLimit(concurrencyLimit)

	for key, aggQuery := range explainResult.Groups.AggregateQueries {
		eg.Go(func() error {
			return qce.queryGroupAggregate(ctx, explainResult, results, key, aggQuery)
		})
	}

	if err := eg.Wait(); err != nil {
		return nil, err
	}

	return results.groups, nil
}

func (qce *QueryCollectionExecutor) queryGroupAggregate(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
	results *GroupResults,
	key string,
	query string,
) error {
	rawMatrix, err := qce.queryRangeMatrix(ctx, query, explainResult.Request)
	if err != nil {
		return err
	}

	results.MergeMatrixToAggregateGroups(
		rawMatrix,
		key,
		explainResult.Groups.Dimensions,
	)

	return nil
}

// AggregateResults store the aggregate results with mutex lock.
type AggregateResults struct {
	results map[string]any
	lock    sync.Mutex
}

// NewAggregateResults create a new AggregateResults.
func NewAggregateResults() *AggregateResults {
	return &AggregateResults{
		results: make(map[string]any),
	}
}

// SetResult sets the result.
func (ar *AggregateResults) SetResult(key string, value any) {
	ar.lock.Lock()
	defer ar.lock.Unlock()

	ar.results[key] = value
}

// GroupResults store the aggregate group results with mutex lock.
type GroupResults struct {
	groups  []schema.Group
	runtime *metadata.RuntimeSettings
	lock    sync.Mutex
}

// NewGroupResults create a new GroupResults.
func NewGroupResults(runtime *metadata.RuntimeSettings) *GroupResults {
	return &GroupResults{
		groups:  []schema.Group{},
		runtime: runtime,
	}
}

// MergeMatrixToAggregateGroups merges matrix results to aggregate groups.
func (gr *GroupResults) MergeMatrixToAggregateGroups(
	rawMatrix model.Matrix,
	key string,
	dimensions []string,
) []schema.Group {
	gr.lock.Lock()
	defer gr.lock.Unlock()

	for _, sample := range rawMatrix {
		gr.groups = gr.mergeSampleToAggregateGroup(gr.groups, sample, key, dimensions)
	}

	return gr.groups
}

func (gr *GroupResults) mergeSampleToAggregateGroup(
	source []schema.Group,
	samples *model.SampleStream,
	key string,
	dimensions []string,
) []schema.Group {
L:
	for _, item := range samples.Values {
		group := schema.Group{
			Dimensions: make([]any, len(dimensions)),
			Aggregates: make(schema.GroupAggregates),
		}

		for i, dim := range dimensions {
			if dim == metadata.TimestampKey {
				group.Dimensions[i] = formatTimestamp(item.Timestamp, gr.runtime.Format.Timestamp)

				continue
			}

			group.Dimensions[i] = samples.Metric[model.LabelName(dim)]
		}

		for i, g := range source {
			if equalSlice(g.Dimensions, group.Dimensions) {
				source[i].Aggregates[key] = item.Value

				continue L
			}
		}

		group.Aggregates[key] = formatValue(item.Value, gr.runtime.Format)
		source = append(source, group)
	}

	return source
}
