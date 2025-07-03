package internal

import (
	"fmt"
	"regexp"
	"slices"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// LabelExpressionField the structured data of a label field expression.
type LabelExpressionField struct {
	Value   string
	IsRegex bool
}

// LabelExpression the structured data of a label expression.
type LabelExpression struct {
	Name        string
	Expressions []schema.ExpressionBinaryComparisonOperator
}

type LabelExpressionBuilder struct {
	LabelExpression

	includes []LabelExpressionField
	excludes map[LabelExpressionField]*regexp.Regexp
}

// Evaluate evaluates the list of expressions and returns the query string.
func (le *LabelExpressionBuilder) Evaluate(variables map[string]any) (string, bool, error) {
	if len(le.Expressions) == 0 {
		return "", true, nil
	}

	le.includes = []LabelExpressionField{}
	le.excludes = map[LabelExpressionField]*regexp.Regexp{}

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
		if le.excludeField(inc) {
			continue
		}

		includes = append(includes, inc.Value)
		isIncludeRegex = isIncludeRegex || inc.IsRegex
	}

	if (len(le.includes) > 0 && len(includes) == 0) ||
		(len(includes) == 0 && len(le.excludes) == 0) {
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

func (le *LabelExpressionBuilder) excludeField(inc LabelExpressionField) bool {
	for exc, erg := range le.excludes {
		if (erg == nil && inc.Value == exc.Value) ||
			(erg != nil && erg.MatchString(inc.Value)) {
			delete(le.excludes, exc)

			return true
		}
	}

	return false
}

func (le *LabelExpressionBuilder) evalLabelComparison(operator string, value any) (bool, error) {
	switch operator {
	case metadata.Equal, metadata.Regex, metadata.Contains, metadata.ContainsInsensitive, metadata.StartsWith, metadata.StartsWithInsensitive, metadata.EndsWith, metadata.EndsWithInsensitive:
		return le.evalLabelComparisonRegex(operator, value)
	case metadata.In:
		return le.evalLabelComparisonIn(value)
	case metadata.NotEqual, metadata.NotRegex:
		return le.evalLabelComparisonNotMatch(operator, value)
	case metadata.NotIn:
		return le.evalLabelComparisonNotIn(value)
	default:
		return false, fmt.Errorf("unsupported comparison operator `%s`", operator)
	}
}

func (le *LabelExpressionBuilder) evalLabelComparisonRegex(
	operator string,
	value any,
) (bool, error) {
	strValuePtr, err := utils.DecodeNullableString(value)
	if err != nil {
		return false, err
	}

	if strValuePtr == nil {
		return true, nil
	}

	strValue := *strValuePtr
	isRegex := operator != metadata.Equal

	switch operator {
	case metadata.Contains:
		strValue = "(.*" + strValue + ".*)"
	case metadata.ContainsInsensitive:
		strValue = "((?i).*" + strValue + ".*)"
	case metadata.StartsWith:
		strValue = "(^" + strValue + ".*)"
	case metadata.StartsWithInsensitive:
		strValue = "((?i)^" + strValue + ".*)"
	case metadata.EndsWith:
		strValue = "(.*" + strValue + "$)"
	case metadata.EndsWithInsensitive:
		strValue = "((?i).*" + strValue + "$)"
	}

	if len(le.includes) == 0 {
		le.includes = []LabelExpressionField{
			{
				Value:   strValue,
				IsRegex: isRegex,
			},
		}

		return true, nil
	}

	var includes []LabelExpressionField

	for _, inc := range le.includes {
		if inc.Value == strValue {
			includes = append(includes, LabelExpressionField{
				Value:   inc.Value,
				IsRegex: inc.IsRegex && isRegex,
			})

			continue
		}

		if isRegex {
			rg, err := regexp.Compile(strValue)
			if err != nil {
				return false, fmt.Errorf("invalid regular expression `%s`: %w", strValue, err)
			}

			if rg.MatchString(inc.Value) {
				includes = append(includes, inc)
			}
		}
	}

	if len(includes) == 0 {
		return false, nil
	}

	le.includes = includes

	return true, nil
}

func (le *LabelExpressionBuilder) evalLabelComparisonIn(value any) (bool, error) {
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

	return true, nil
}

func (le *LabelExpressionBuilder) evalLabelComparisonNotMatch(
	operator string,
	value any,
) (bool, error) {
	strValue, err := utils.DecodeNullableString(value)
	if err != nil {
		return false, err
	}

	if strValue == nil {
		return true, nil
	}

	isRegex := operator == metadata.NotRegex

	var rg *regexp.Regexp

	if isRegex {
		rg, err = regexp.Compile(*strValue)
		if err != nil {
			return false, fmt.Errorf("invalid regular expression `%s`: %w", *strValue, err)
		}
	}

	le.excludes[LabelExpressionField{
		Value:   *strValue,
		IsRegex: isRegex,
	}] = rg

	return true, nil
}

func (le *LabelExpressionBuilder) evalLabelComparisonNotIn(value any) (bool, error) {
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
		}] = nil
	}

	return true, nil
}
