package internal

import (
	"context"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/prometheus/common/model"
)

func (qce *QueryCollectionExecutor) mergeMatrixToAggregateGroups(source []schema.Group, rawMatrix model.Matrix, key string, dimensions []string) []schema.Group {
	for _, sample := range rawMatrix {
		source = qce.mergeSampleToAggregateGroup(source, sample, key, dimensions)
	}

	return source
}

func (qce *QueryCollectionExecutor) mergeSampleToAggregateGroup(source []schema.Group, samples *model.SampleStream, key string, dimensions []string) []schema.Group {
L:
	for _, item := range samples.Values {
		group := schema.Group{
			Dimensions: make([]any, len(dimensions)),
			Aggregates: make(schema.GroupAggregates),
		}

		for i, dim := range dimensions {
			if dim == metadata.TimestampKey {
				group.Dimensions[i] = formatTimestamp(item.Timestamp, qce.Runtime.Format.Timestamp)

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

		group.Aggregates[key] = formatValue(item.Value, qce.Runtime.Format)
		source = append(source, group)
	}

	return source
}

func (qce *QueryCollectionExecutor) queryGroupAggregates(
	ctx context.Context,
	explainResult *QueryCollectionExplainResult,
) ([]schema.Group, error) {
	if explainResult.Groups == nil {
		return nil, nil
	}

	results := []schema.Group{}

	for key, aggQuery := range explainResult.Groups.AggregateQueries {
		rawMatrix, err := qce.queryRangeMatrix(ctx, aggQuery, explainResult.Request)
		if err != nil {
			return nil, err
		}

		results = qce.mergeMatrixToAggregateGroups(results, rawMatrix, key, explainResult.Groups.Dimensions)
	}

	return results, nil
}
