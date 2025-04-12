package internal

import (
	"errors"
	"fmt"
	"regexp"
	"slices"
	"strings"
	"time"

	"github.com/hasura/ndc-prometheus/connector/metadata"
)

var errEmptyResult = errors.New("empty result")

var rangeVectorFunctionNames = []metadata.PromQLFunctionName{
	metadata.AbsentOverTime,
	metadata.Changes,
	metadata.Delta,
	metadata.Derivative,
	metadata.HoltWinters,
	metadata.IDelta,
	metadata.Increase,
	metadata.IRate,
	metadata.PredictLinear,
	metadata.Rate,
	metadata.Resets,
	metadata.MaxOverTime,
	metadata.MinOverTime,
	metadata.AvgOverTime,
	metadata.CountOverTime,
	metadata.LastOverTime,
	metadata.MadOverTime,
	metadata.PresentOverTime,
	metadata.StddevOverTime,
	metadata.StdvarOverTime,
	metadata.SumOverTime,
}

var labelBinaryOperators = map[metadata.LabelComparisonOperator]string{
	metadata.LabelEqual:    "=",
	metadata.LabelNotEqual: "!=",
	metadata.LabelRegex:    "=~",
	metadata.LabelNotRegex: "!~",
}

var valueBinaryOperators = map[metadata.ComparisonOperator]string{
	metadata.Equal:          "==",
	metadata.NotEqual:       "!=",
	metadata.Less:           "<",
	metadata.LessOrEqual:    "<=",
	metadata.Greater:        ">",
	metadata.GreaterOrEqual: ">=",
}

type PromQLExpressionContext struct {
	Runtime    *metadata.RuntimeSettings
	MetricName string
	Functions  []PromQLFunction
	Value      *ValueComparison
}

type PromQLExpression interface {
	Render(ctx PromQLExpressionContext) (string, error)
}

type PromQLExpressionAnd struct {
	Expressions []PromQLExpression
}

func (expr PromQLExpressionAnd) Render(ctx PromQLExpressionContext) (string, error) {
	return renderPromQLExpressions(ctx, " and ", expr.Expressions)
}

type PromQLExpressionOr struct {
	Expressions []PromQLExpression
}

func (expr PromQLExpressionOr) Render(ctx PromQLExpressionContext) (string, error) {
	return renderPromQLExpressions(ctx, " or ", expr.Expressions)
}

func renderPromQLExpressions(ctx PromQLExpressionContext, operator string, expressions []PromQLExpression) (string, error) {
	exprLen := len(expressions)

	switch exprLen {
	case 0:
		return "", nil
	case 1:
		return expressions[0].Render(ctx)
	default:
		statements := []string{}

		for _, expr := range expressions {
			result, err := expr.Render(ctx)
			if err != nil {
				return "", err
			}

			if result == "" {
				continue
			}

			statements = append(statements, "("+result+")")
		}

		return strings.Join(statements, operator), nil
	}
}

type PromQLExpressionBlock struct {
	Labels map[string]LabelComparisons
	Value  *ValueComparison
	Offset time.Duration
}

func (expr PromQLExpressionBlock) Render(ctx PromQLExpressionContext) (string, error) {
	var rangeVectorFunction *PromQLFunction
	var err error

	functions := ctx.Functions

	if len(functions) > 0 && slices.Contains(rangeVectorFunctionNames, ctx.Functions[0].Name) {
		rangeVectorFunction = &ctx.Functions[0]
		functions = ctx.Functions[1:]
	}

	query := ctx.MetricName + expr.renderLabelConditions()

	if rangeVectorFunction != nil {
		query, err = rangeVectorFunction.Render(query, expr.Offset, ctx.Runtime)
		if err != nil {
			return "", err
		}
	}

	if expr.Value != nil {
		op, ok := valueBinaryOperators[expr.Value.Operator]
		if !ok {
			return "", fmt.Errorf("value: unsupported comparison operator `%s`", expr.Value.Operator)
		}

		query += fmt.Sprintf(" %s %f", op, expr.Value.Value)
	}

	for _, fn := range functions {
		query, err = fn.Render(query, 0, ctx.Runtime)
		if err != nil {
			return "", err
		}
	}

	return query, nil
}

func (expr PromQLExpressionBlock) renderLabelConditions() string {
	if len(expr.Labels) == 0 {
		return ""
	}

	conditions := []string{}

	for name, exprs := range expr.Labels {
		cond := exprs.Render(name)
		if cond != "" {
			conditions = append(conditions, cond)
		}
	}

	if len(conditions) == 0 {
		return ""
	}

	var sb strings.Builder
	sb.WriteRune('{')

	for i, cond := range conditions {
		if i > 0 {
			sb.WriteRune(',')
		}

		sb.WriteString(cond)
	}

	sb.WriteRune('}')

	return sb.String()
}

// LabelComparisons is the alias of the label comparison list.
type LabelComparisons []LabelComparison

// Add tries to add and merge comparisons
func (lcs *LabelComparisons) Add(value LabelComparison) (bool, error) {
	if len(*lcs) == 0 {
		*lcs = append(*lcs, value)
	}

	results := []LabelComparison{value}

	for _, target := range *lcs {
		resultsLen := len(results)

		mergedResults, err := mergeLabelComparisonPair(target, results[resultsLen-1])
		if err != nil {
			return false, err
		}

		if len(mergedResults) == 0 {
			return false, nil
		}

		results = append(results[:resultsLen-1], mergedResults...)
	}

	*lcs = results

	return true, nil
}

// Render builds the condition string.
func (lcs LabelComparisons) Render(name string) string {
	switch len(lcs) {
	case 0:
		return ""
	case 1:
		return name + lcs[0].String()
	default:
		conditions := []string{}

		for _, comp := range lcs {
			value := comp.PromQLValue()
			if value == "" {
				continue
			}

			conditions = append(conditions, value)
		}

		if len(conditions) == 0 {
			return ""
		}

		var sb strings.Builder
		sb.WriteString(name)

		if lcs[0].Operator.IsNegative() {
			sb.WriteString(labelBinaryOperators[metadata.LabelNotRegex])
		} else {
			sb.WriteString(labelBinaryOperators[metadata.LabelRegex])
		}

		sb.WriteRune('"')

		for i, cond := range conditions {
			if i > 0 {
				sb.WriteRune('|')
			}

			sb.WriteString(cond)
		}

		sb.WriteRune('"')

		return sb.String()
	}
}

// LabelComparison the structured data of a label field expression.
type LabelComparison struct {
	Value    []string
	Operator metadata.LabelComparisonOperator
}

// Validate checks if the input value is valid with the operator.
func (lc LabelComparison) Validate(value string) (bool, error) {
	switch lc.Operator {
	case metadata.LabelEqual:
		return lc.Value[0] == value, nil
	case metadata.LabelNotEqual:
		return lc.Value[0] != value, nil
	case metadata.LabelContains:
		return strings.Contains(value, lc.Value[0]), nil
	case metadata.LabelNotContains:
		return !strings.Contains(value, lc.Value[0]), nil
	case metadata.LabelContainsInsensitive:
		return strings.Contains(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelNotContainsInsensitive:
		return !strings.Contains(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelStartsWith:
		return strings.HasPrefix(value, lc.Value[0]), nil
	case metadata.LabelNotStartsWith:
		return !strings.HasPrefix(value, lc.Value[0]), nil
	case metadata.LabelStartsWithInsensitive:
		return strings.HasPrefix(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelNotStartsWithInsensitive:
		return !strings.HasPrefix(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelEndsWith:
		return strings.HasSuffix(value, lc.Value[0]), nil
	case metadata.LabelNotEndsWith:
		return !strings.HasSuffix(value, lc.Value[0]), nil
	case metadata.LabelEndsWithInsensitive:
		return strings.HasSuffix(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelNotEndsWithInsensitive:
		return !strings.HasSuffix(strings.ToLower(value), lc.Value[0]), nil
	case metadata.LabelIn:
		return slices.Contains(lc.Value, value), nil
	case metadata.LabelNotIn:
		return !slices.Contains(lc.Value, value), nil
	case metadata.LabelRegex:
		pattern, err := regexp.Compile(lc.Value[0])
		if err != nil {
			return false, err
		}

		return pattern.MatchString(value), nil
	case metadata.LabelNotRegex:
		pattern, err := regexp.Compile(lc.Value[0])
		if err != nil {
			return false, err
		}

		return !pattern.MatchString(value), nil
	default:
		return false, fmt.Errorf("invalid label comparison operator %s", lc.Operator)
	}
}

// PromQLOperator returns the promQL operator string
func (lc LabelComparison) PromQLOperator() string {
	switch lc.Operator {
	case metadata.LabelEqual:
		return labelBinaryOperators[metadata.LabelEqual]
	case metadata.LabelNotEqual:
		return labelBinaryOperators[metadata.LabelNotEqual]
	case metadata.LabelNotContains, metadata.LabelNotContainsInsensitive, metadata.LabelNotEndsWithInsensitive, metadata.LabelNotEndsWith, metadata.LabelNotRegex, metadata.LabelNotStartsWith, metadata.LabelNotIn, metadata.LabelNotStartsWithInsensitive:
		return labelBinaryOperators[metadata.LabelNotRegex]
	default:
		return labelBinaryOperators[metadata.LabelRegex]
	}
}

// PromQLValue returns the promQL-serialized string.
func (lc LabelComparison) PromQLValue() string {
	if len(lc.Value) == 0 {
		return ""
	}

	switch lc.Operator {
	case metadata.LabelEqual, metadata.LabelNotEqual, metadata.LabelRegex, metadata.LabelNotRegex, metadata.LabelContains, metadata.LabelNotContains:
		return lc.Value[0]
	case metadata.LabelContainsInsensitive, metadata.LabelNotContainsInsensitive:
		return "((?i)" + lc.Value[0] + ")"
	case metadata.LabelStartsWith, metadata.LabelNotStartsWith:
		return "^" + lc.Value[0]
	case metadata.LabelStartsWithInsensitive, metadata.LabelNotStartsWithInsensitive:
		return "(^(?i)" + lc.Value[0] + ")"
	case metadata.LabelEndsWith, metadata.LabelNotEndsWith:
		return lc.Value[0] + "$"
	case metadata.LabelEndsWithInsensitive, metadata.LabelNotEndsWithInsensitive:
		return "((?i)" + lc.Value[0] + "$)"
	default:
		return strings.Join(lc.Value, "|")
	}
}

// String implements the fmt.Stringer interface.
func (lc LabelComparison) String() string {
	return lc.PromQLOperator() + `"` + lc.PromQLValue() + `"`
}

// ValueComparison the structured data of a value field expression.
type ValueComparison struct {
	Value    float64
	Operator metadata.ComparisonOperator
}

// try merging expressions of the same label to reduce the complexity.
func mergeLabelComparisonPair(a, b LabelComparison) ([]LabelComparison, error) {
	v1 := a
	v2 := b

	if slices.Contains([]metadata.LabelComparisonOperator{metadata.LabelIn, metadata.LabelEqual, metadata.LabelNotEqual, metadata.LabelNotIn}, b.Operator) {
		v1 = b
		v2 = a
	}

	switch v1.Operator {
	case metadata.LabelEqual:
		matched, err := v2.Validate(v1.Value[0])
		if err != nil {
			return nil, err
		}

		if !matched {
			return nil, errEmptyResult
		}

		return []LabelComparison{v1}, nil
	case metadata.LabelNotEqual:
		matched, err := v2.Validate(v1.Value[0])
		if err != nil {
			return nil, err
		}

		if matched {
			return nil, errEmptyResult
		}

		return nil, nil
	case metadata.LabelIn:
		var values []string

		for _, item := range v1.Value {
			matched, err := v2.Validate(item)
			if err != nil {
				return nil, err
			}

			if matched {
				values = append(values, item)
			}
		}

		if len(values) == 0 {
			return nil, errEmptyResult
		}

		return []LabelComparison{
			{
				Operator: metadata.LabelIn,
				Value:    values,
			},
		}, nil
	default:
		return nil, nil
	}
}
