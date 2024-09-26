package internal

import (
	"encoding/json"
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

var pgArrayStringRegex = regexp.MustCompile(`^{([\w-,]+)}$`)

// LabelExpressionField the structured data of a label field expression
type LabelExpressionField struct {
	Value   string
	IsRegex bool
}

// LabelExpression the structured data of a label expression
type LabelExpression struct {
	Name        string
	Expressions []schema.ExpressionBinaryComparisonOperator
}

type LabelExpressionBuilder struct {
	LabelExpression

	includes []LabelExpressionField
	excludes map[LabelExpressionField]bool
}

// Evaluate evaluates the list of expressions and returns the query string
func (le *LabelExpressionBuilder) Evaluate(variables map[string]any) (string, bool, error) {
	if len(le.Expressions) == 0 {
		return "", true, nil
	}
	le.includes = []LabelExpressionField{}
	le.excludes = map[LabelExpressionField]bool{}
	for _, expr := range le.Expressions {
		value, err := getComparisonValue(expr.Value, variables)
		if err != nil {
			return "", false, err
		}
		ok, err := le.evalLabelComparison(expr.Operator, value)
		if err != nil || !ok {
			return "", false, err
		}
	}

	var isIncludeRegex bool
	includes := []string{}
	for _, inc := range le.includes {
		if _, ok := le.excludes[inc]; ok {
			delete(le.excludes, inc)
			continue
		}
		includes = append(includes, inc.Value)
		isIncludeRegex = isIncludeRegex || inc.IsRegex
	}
	if len(includes) == 0 && len(le.excludes) == 0 {
		// all equal and not-equal labels are matched together,
		// so the result is always empty
		return "", false, nil
	}

	// if the label equals A or B but not C => equals A or B
	if len(includes) > 0 {
		operator := "="
		if len(includes) > 1 || isIncludeRegex {
			operator = "=~"
		}
		return fmt.Sprintf(`%s%s"%s"`, le.Name, operator, strings.Join(includes, "|")), true, nil
	}

	// exclude only
	var isExcludeRegex bool
	excludes := make([]string, 0, len(le.excludes))
	for ev := range le.excludes {
		excludes = append(excludes, ev.Value)
		isExcludeRegex = isExcludeRegex || ev.IsRegex
	}

	slices.Sort(excludes)
	operator := "!="
	if len(excludes) > 1 || isExcludeRegex {
		operator = "!~"
	}
	return fmt.Sprintf(`%s%s"%s"`, le.Name, operator, strings.Join(excludes, "|")), true, nil
}

func (le *LabelExpressionBuilder) evalLabelComparison(operator string, value any) (bool, error) {
	switch operator {
	case metadata.Equal, metadata.Regex:
		strValue, err := utils.DecodeNullableString(value)
		if err != nil {
			return false, err
		}
		if strValue == nil {
			return true, nil
		}
		if len(le.includes) > 0 && !slices.ContainsFunc(le.includes, func(f LabelExpressionField) bool {
			return f.Value == *strValue
		}) {
			return false, nil
		}
		le.includes = []LabelExpressionField{
			{
				Value:   *strValue,
				IsRegex: operator == metadata.Regex,
			},
		}
	case metadata.In:
		strValues, err := decodeStringSlice(value)
		if err != nil {
			return false, err
		}
		if strValues == nil {
			return true, nil
		}
		if len(strValues) == 0 {
			return false, nil
		}
		newValues := make([]LabelExpressionField, len(strValues))
		for i, v := range strValues {
			newValues[i] = LabelExpressionField{
				Value: v,
			}
		}
		if len(le.includes) == 0 {
			le.includes = newValues
			return true, nil
		}
		le.includes = intersection(le.includes, newValues)
		if len(le.includes) == 0 {
			return false, nil
		}
	case metadata.NotEqual, metadata.NotRegex:
		strValue, err := utils.DecodeNullableString(value)
		if err != nil {
			return false, err
		}
		if strValue == nil {
			return true, nil
		}

		le.excludes[LabelExpressionField{
			Value:   *strValue,
			IsRegex: operator == metadata.NotRegex,
		}] = true
	case metadata.NotIn:
		strValues, err := decodeStringSlice(value)
		if err != nil {
			return false, err
		}
		if strValues == nil {
			return true, nil
		}
		for _, v := range strValues {
			le.excludes[LabelExpressionField{
				Value: v,
			}] = true
		}
	default:
		return false, fmt.Errorf("unsupported comparison operator `%s`", operator)
	}

	return true, nil
}

func decodeStringSlice(value any) ([]string, error) {
	if utils.IsNil(value) {
		return nil, nil
	}
	var err error
	sliceValue := []string{}
	if str, ok := value.(string); ok {
		matches := pgArrayStringRegex.FindStringSubmatch(str)
		if len(matches) > 1 {
			sliceValue = strings.Split(matches[1], ",")
		} else {
			// try to parse the slice from the json string
			err = json.Unmarshal([]byte(str), &sliceValue)
		}
	} else {
		sliceValue, err = utils.DecodeStringSlice(value)
	}
	if err != nil {
		return nil, err
	}

	return sliceValue, nil
}

func intersection[T comparable](sliceA []T, sliceB []T) []T {
	var result []T
	if len(sliceA) == 0 || len(sliceB) == 0 {
		return result
	}

	for _, a := range sliceA {
		if slices.Contains(sliceB, a) {
			result = append(result, a)
		}
	}

	return result
}
