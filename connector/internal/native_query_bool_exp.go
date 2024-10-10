package internal

import (
	"fmt"
	"regexp"
	"slices"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
)

// NativeQueryLabelBoolExp represents the boolean expression object type
type NativeQueryLabelBoolExp struct {
	Equal    *string
	NotEqual *string
	In       []string
	NotIn    []string
	Regex    *regexp.Regexp
	NotRegex *regexp.Regexp
}

// Validate validates the value
func (be NativeQueryLabelBoolExp) Validate(value string) bool {
	return (be.Equal == nil || *be.Equal == value) &&
		(be.NotEqual == nil || *be.NotEqual != value) &&
		(be.In == nil || slices.Contains(be.In, value)) &&
		(be.NotIn == nil || !slices.Contains(be.NotIn, value)) &&
		(be.Regex == nil || be.Regex.MatchString(value)) &&
		(be.NotRegex == nil || !be.NotRegex.MatchString(value))
}

// FromValue decode any value to NativeQueryLabelBoolExp.
func (be *NativeQueryLabelBoolExp) FromValue(value any) error {
	valueMap, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid boolean expression argument %v", value)
	}
	if len(valueMap) == 0 {
		return nil
	}

	var err error
	be.Equal, err = utils.GetNullableString(valueMap, metadata.Equal)
	if err != nil {
		return err
	}
	be.NotEqual, err = utils.GetNullableString(valueMap, metadata.NotEqual)
	if err != nil {
		return err
	}
	rawRegex, err := utils.GetNullableString(valueMap, metadata.Regex)
	if err != nil {
		return err
	}

	if rawRegex != nil {
		regex, err := regexp.Compile(*rawRegex)
		if err != nil {
			return fmt.Errorf("invalid _regex: %s", err)
		}
		be.Regex = regex
	}

	rawNotRegex, err := utils.GetNullableString(valueMap, metadata.NotRegex)
	if err != nil {
		return err
	}

	if rawNotRegex != nil {
		nregex, err := regexp.Compile(*rawNotRegex)
		if err != nil {
			return fmt.Errorf("invalid _nregex: %s", err)
		}
		be.NotRegex = nregex
	}

	if v, ok := valueMap[metadata.In]; ok {
		be.In, err = decodeStringSlice(v)
		if err != nil {
			return err
		}
	}
	if v, ok := valueMap[metadata.NotIn]; ok {
		be.NotIn, err = decodeStringSlice(v)
		if err != nil {
			return err
		}
	}

	return nil
}

func (nqe *NativeQueryExecutor) filterVectorResults(vector model.Vector, where map[string]NativeQueryLabelBoolExp) model.Vector {
	if len(where) == 0 || len(vector) == 0 {
		return vector
	}
	results := model.Vector{}
	for _, item := range vector {
		if nqe.validateLabelBoolExp(item.Metric, where) {
			results = append(results, item)
		}
	}
	return results
}

func (nqe *NativeQueryExecutor) filterMatrixResults(matrix model.Matrix, where map[string]NativeQueryLabelBoolExp) model.Matrix {
	if len(where) == 0 || len(matrix) == 0 {
		return matrix
	}
	results := model.Matrix{}
	for _, item := range matrix {
		if nqe.validateLabelBoolExp(item.Metric, where) {
			results = append(results, item)
		}
	}
	return results
}

func (nqe *NativeQueryExecutor) validateLabelBoolExp(labels model.Metric, where map[string]NativeQueryLabelBoolExp) bool {
	for key, boolExp := range where {
		if labelValue, ok := labels[model.LabelName(key)]; ok {
			if !boolExp.Validate(string(labelValue)) {
				return false
			}
		}
	}
	return true
}

func decodeNativeQueryLabelBoolExps(value any) (map[string]NativeQueryLabelBoolExp, error) {
	results := make(map[string]NativeQueryLabelBoolExp)
	if utils.IsNil(value) {
		return results, nil
	}

	valueMap, ok := value.(map[string]any)
	if !ok {
		return nil, fmt.Errorf("invalid where; expected map, got: %v", value)
	}

	for k, v := range valueMap {
		boolExp := NativeQueryLabelBoolExp{}
		if err := boolExp.FromValue(v); err != nil {
			return nil, err
		}
		results[k] = boolExp
	}

	return results, nil
}
