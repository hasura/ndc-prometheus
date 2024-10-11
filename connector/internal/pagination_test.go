package internal

import (
	"fmt"
	"math/rand/v2"
	"strings"
	"testing"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
)

func TestSortAndPaginateVector(t *testing.T) {
	now := model.Now()
	vectorFixtures := model.Vector{}
	for i := 0; i < 100; i++ {
		vectorFixtures = append(vectorFixtures, &model.Sample{
			Metric: model.Metric{
				"job":      "ndc-prometheus",
				"instance": "ndc-prometheus:8080",
			},
			Timestamp: now.Add(time.Duration(i) * time.Minute),
			Value:     model.SampleValue(float64(rand.IntN(100)) / 100),
		})
	}

	results := append(model.Vector{}, vectorFixtures...)
	sortVector(results, []ColumnOrder{{
		Name:       metadata.ValueKey,
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i-1].Value <= results[i].Value)
	}

	sortVector(results, []ColumnOrder{{
		Name:       metadata.ValueKey,
		Descending: true,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i-1].Value >= results[i].Value)
	}

	sortVector(results, []ColumnOrder{{
		Name:       metadata.TimestampKey,
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i-1].Timestamp < results[i].Timestamp)
	}

	sortVector(results, []ColumnOrder{{
		Name:       "job",
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i-1].Metric["job"] == results[i].Metric["job"])
	}

	assert.DeepEqual(t, paginateVector(results, schema.Query{
		Offset: utils.ToPtr(50),
		Limit:  utils.ToPtr(1),
	})[0], results[50])
}

func TestSortAndPaginateMatrix(t *testing.T) {
	now := model.Now()
	matrixFixtures := model.Matrix{}
	for i := 0; i < 10; i++ {
		matrixFixtures = append(matrixFixtures, &model.SampleStream{
			Metric: model.Metric{
				"job":      "ndc-prometheus",
				"instance": model.LabelValue(fmt.Sprintf("ndc-prometheus:%d", i)),
			},
			Values: []model.SamplePair{
				{
					Timestamp: now.Add(time.Duration(i) * time.Minute),
					Value:     model.SampleValue(float64(rand.IntN(100)) / 100),
				},
				{
					Timestamp: now.Add(time.Duration(i+1) * time.Minute),
					Value:     model.SampleValue(float64(rand.IntN(100)) / 100),
				},
			},
		})
	}

	results := append(model.Matrix{}, matrixFixtures...)
	sortMatrix(results, []ColumnOrder{{
		Name:       metadata.ValueKey,
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i].Values[0].Value <= results[i].Values[1].Value)
	}

	sortMatrix(results, []ColumnOrder{{
		Name:       metadata.ValueKey,
		Descending: true,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i].Values[0].Value >= results[i].Values[1].Value)
	}

	sortMatrix(results, []ColumnOrder{{
		Name:       metadata.TimestampKey,
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, results[i].Values[0].Timestamp < results[i].Values[1].Timestamp)
	}

	sortMatrix(results, []ColumnOrder{{
		Name:       "instance",
		Descending: false,
	}})

	for i := 1; i < len(results); i++ {
		assert.Assert(t, strings.Compare(string(results[i-1].Metric["instance"]), string(results[i].Metric["instance"])) == -1)
	}

	mapResults := createGroupQueryResultsFromMatrix(results, map[string]metadata.LabelInfo{
		"job":      {},
		"instance": {},
	}, &metadata.RuntimeSettings{})
	assert.DeepEqual(t, paginateQueryResults(mapResults, schema.Query{
		Offset: utils.ToPtr(5),
		Limit:  utils.ToPtr(1),
	})[0], mapResults[5])
}
