package internal

import (
	"fmt"
	"math/rand/v2"
	"testing"
	"time"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/prometheus/common/model"
	"gotest.tools/v3/assert"
)

func TestFilterVectorResults(t *testing.T) {
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

	nqe := &NativeQueryExecutor{}
	results, err := nqe.filterVectorResults(vectorFixtures, nil)
	assert.NilError(t, err)
	assert.DeepEqual(t, results, vectorFixtures)

	testCases := []struct {
		Name       string
		Expression schema.Expression
		Expected   model.Vector
	}{
		{
			Name: "equal",
			Expression: schema.NewExpressionAnd(
				schema.NewExpressionBinaryComparisonOperator(
					*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
					"_eq",
					schema.NewComparisonValueScalar("foo"),
				),
			).Encode(),
			Expected: model.Vector{},
		},
		{
			Name: "is_null",
			Expression: schema.NewExpressionAnd(
				schema.NewExpressionUnaryComparisonOperator(
					*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
					schema.UnaryComparisonOperatorIsNull,
				),
			).Encode(),
			Expected: model.Vector{},
		},
		{
			Name: "neq",
			Expression: schema.NewExpressionAnd(
				schema.NewExpressionBinaryComparisonOperator(
					*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
					"_neq",
					schema.NewComparisonValueScalar("foo"),
				),
			).Encode(),
			Expected: vectorFixtures,
		},
		{
			Name: "in",
			Expression: schema.NewExpressionAnd(
				schema.NewExpressionBinaryComparisonOperator(
					*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
					"_in",
					schema.NewComparisonValueScalar([]string{"ndc-prometheus"}),
				),
			).Encode(),
			Expected: vectorFixtures,
		},
		{
			Name: "nin",
			Expression: schema.NewExpressionAnd(
				schema.NewExpressionBinaryComparisonOperator(
					*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
					"_nin",
					schema.NewComparisonValueScalar([]string{"foo"}),
				),
			).Encode(),
			Expected: vectorFixtures,
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			results, err = nqe.filterVectorResults(vectorFixtures, tc.Expression)
			assert.NilError(t, err)
			assert.DeepEqual(t, results, tc.Expected)
		})
	}
}

func TestFilterMatrixResults(t *testing.T) {
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

	nqe := &NativeQueryExecutor{}
	results, err := nqe.filterMatrixResults(matrixFixtures, &NativeQueryRequest{})
	assert.NilError(t, err)
	assert.DeepEqual(t, results, matrixFixtures)

	testCases := []struct {
		Name     string
		Request  *NativeQueryRequest
		Expected model.Matrix
	}{
		{
			Name: "or",
			Request: &NativeQueryRequest{
				Expression: schema.NewExpressionOr(
					schema.NewExpressionBinaryComparisonOperator(
						*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
						"_eq",
						schema.NewComparisonValueScalar("foo"),
					),
				).Encode(),
			},
			Expected: model.Matrix{},
		},
		{
			Name: "value",
			Request: &NativeQueryRequest{
				HasValueBoolExp: true,
				Expression: schema.NewExpressionOr(
					schema.NewExpressionBinaryComparisonOperator(
						*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
						"_regex",
						schema.NewComparisonValueScalar("foo"),
					),
					schema.NewExpressionBinaryComparisonOperator(
						*schema.NewComparisonTargetColumn("value", []string{}, []schema.PathElement{}),
						"_gte",
						schema.NewComparisonValueScalar(0),
					),
				).Encode(),
			},
			Expected: matrixFixtures,
		},
		{
			Name: "not",
			Request: &NativeQueryRequest{
				HasValueBoolExp: true,
				Expression: schema.NewExpressionNot(schema.NewExpressionOr(
					schema.NewExpressionBinaryComparisonOperator(
						*schema.NewComparisonTargetColumn("job", []string{}, []schema.PathElement{}),
						"_nregex",
						schema.NewComparisonValueScalar("foo"),
					),
					schema.NewExpressionBinaryComparisonOperator(
						*schema.NewComparisonTargetColumn("value", []string{}, []schema.PathElement{}),
						"_lte",
						schema.NewComparisonValueScalar(0),
					),
				)).Encode(),
			},
			Expected: model.Matrix{},
		},
	}

	for _, tc := range testCases {
		t.Run(tc.Name, func(t *testing.T) {
			results, err = nqe.filterMatrixResults(matrixFixtures, tc.Request)
			assert.NilError(t, err)
			assert.DeepEqual(t, results, tc.Expected)
		})
	}

}
