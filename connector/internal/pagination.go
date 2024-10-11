package internal

import (
	"math"
	"slices"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/prometheus/common/model"
)

func sortVector(vector model.Vector, sortElements []ColumnOrder) {
	if len(sortElements) == 0 {
		return
	}

	slices.SortFunc(vector, func(a *model.Sample, b *model.Sample) int {
		for _, elem := range sortElements {
			iOrder := 1
			if elem.Descending {
				iOrder = -1
			}
			switch elem.Name {
			case metadata.ValueKey:
				if a.Value.Equal(b.Value) {
					continue
				}
				if math.IsNaN(float64(a.Value)) {
					return 1 * iOrder
				}
				if math.IsNaN(float64(b.Value)) {
					return -1 * iOrder
				}
				if a.Value > b.Value {
					return 1 * iOrder
				} else {
					return -1 * iOrder
				}
			case metadata.TimestampKey:
				difference := a.Timestamp.Sub(b.Timestamp)
				if difference == 0 {
					continue
				}
				return int(difference) * iOrder
			default:
				if len(a.Metric) == 0 {
					continue
				}
				labelA, okA := a.Metric[model.LabelName(elem.Name)]
				labelB, okB := b.Metric[model.LabelName(elem.Name)]
				if !okA && !okB {
					continue
				}
				difference := strings.Compare(string(labelA), string(labelB))
				if difference == 0 {
					continue
				}
				return difference * iOrder
			}
		}
		return 0
	})
}

func sortMatrix(matrix model.Matrix, sortElements []ColumnOrder) {
	if len(sortElements) == 0 {
		return
	}

	slices.SortFunc(matrix, func(a *model.SampleStream, b *model.SampleStream) int {
		for _, elem := range sortElements {
			iOrder := 1
			if elem.Descending {
				iOrder = -1
			}
			switch elem.Name {
			case metadata.ValueKey, metadata.TimestampKey:
				sortSamplePair(a.Values, elem.Name, iOrder)
				sortSamplePair(b.Values, elem.Name, iOrder)
			default:
				if len(a.Metric) == 0 {
					continue
				}
				labelA, okA := a.Metric[model.LabelName(elem.Name)]
				labelB, okB := b.Metric[model.LabelName(elem.Name)]
				if !okA && !okB {
					continue
				}
				difference := strings.Compare(string(labelA), string(labelB))
				if difference == 0 {
					continue
				}
				return difference * iOrder
			}
		}
		return 0
	})
}

func sortSamplePair(values []model.SamplePair, key string, iOrder int) {
	slices.SortFunc(values, func(a model.SamplePair, b model.SamplePair) int {
		switch key {
		case metadata.ValueKey:
			if a.Value.Equal(b.Value) {
				return 0
			}
			if math.IsNaN(float64(a.Value)) {
				return 1 * iOrder
			}
			if math.IsNaN(float64(b.Value)) {
				return -1 * iOrder
			}
			if a.Value > b.Value {
				return 1 * iOrder
			} else {
				return -1 * iOrder
			}
		case metadata.TimestampKey:
			return int(a.Timestamp.Sub(b.Timestamp)) * iOrder
		default:
			return 0
		}
	})
}

func paginateVector(vector model.Vector, q schema.Query) model.Vector {
	if q.Offset != nil && *q.Offset > 0 {
		if len(vector) <= *q.Offset {
			return model.Vector{}
		}
		vector = vector[*q.Offset:]
	}
	if q.Limit != nil && *q.Limit < len(vector) {
		vector = vector[:*q.Limit]
	}
	return vector
}

func paginateQueryResults(results []map[string]any, q schema.Query) []map[string]any {
	if q.Offset != nil && *q.Offset > 0 {
		if len(results) <= *q.Offset {
			return []map[string]any{}
		}
		results = results[*q.Offset:]
	}

	if q.Limit != nil && *q.Limit < len(results) {
		results = results[:*q.Limit]
	}

	return results
}
