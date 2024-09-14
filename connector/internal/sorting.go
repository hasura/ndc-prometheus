package internal

import (
	"math"
	"slices"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/prometheus/common/model"
)

func (qce *QueryCollectionExecutor) sortVector(vector model.Vector, sortElements []ColumnOrder) {
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
				return int(float64(a.Value)-float64(b.Value)) * iOrder
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

func (qce *QueryCollectionExecutor) sortMatrix(matrix model.Matrix, sortElements []ColumnOrder) {
	if len(sortElements) == 0 {
		return
	}

	slices.SortFunc(matrix, func(a *model.SampleStream, b *model.SampleStream) int {
		lenA := len(a.Values)
		lenB := len(b.Values)

		for _, elem := range sortElements {
			iOrder := 1
			if elem.Descending {
				iOrder = -1
			}
			switch elem.Name {
			case metadata.ValueKey:
				if lenA == 0 && lenB == 0 {
					continue
				}
				if lenA == 0 {
					return -1 * iOrder
				}
				if lenB == 0 {
					return 1 * iOrder
				}

				valueA := a.Values[lenA-1].Value
				valueB := b.Values[lenB-1].Value

				if valueA.Equal(valueB) {
					continue
				}
				if math.IsNaN(float64(valueA)) {
					return 1 * iOrder
				}
				if math.IsNaN(float64(valueB)) {
					return -1 * iOrder
				}
				return int(float64(valueA)-float64(valueB)) * iOrder
			case metadata.TimestampKey:
				if lenA == 0 && lenB == 0 {
					continue
				}
				if lenA == 0 {
					return -1 * iOrder
				}
				if lenB == 0 {
					return 1 * iOrder
				}

				tsA := a.Values[lenA-1].Timestamp
				tsB := b.Values[lenB-1].Timestamp

				difference := tsA.Sub(tsB)
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
