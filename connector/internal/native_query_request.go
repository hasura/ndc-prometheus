package internal

import (
	"errors"
	"fmt"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
)

// NativeQueryRequest the structured native request which is evaluated from the raw expression
type NativeQueryRequest struct {
	Timestamp  any
	Start      any
	End        any
	Timeout    any
	Step       any
	OrderBy    []ColumnOrder
	Variables  map[string]any
	Expression schema.Expression
}

// EvalNativeQueryRequest evaluates the requested collection data of the query request
func EvalNativeQueryRequest(request *schema.QueryRequest, arguments map[string]any, variables map[string]any) (*NativeQueryRequest, error) {
	result := &NativeQueryRequest{
		Variables: variables,
	}
	if len(request.Query.Predicate) > 0 {
		newExpr, err := result.evalQueryPredicate(request.Query.Predicate)
		if err != nil {
			return nil, err
		}
		if newExpr != nil {
			result.Expression = newExpr.Encode()
		}
	}

	orderBy, err := evalCollectionOrderBy(request.Query.OrderBy)
	if err != nil {
		return nil, err
	}
	result.OrderBy = orderBy
	return result, nil
}

func (pr *NativeQueryRequest) getComparisonValue(input schema.ComparisonValue) (any, error) {
	return getComparisonValue(input, pr.Variables)
}

func (pr *NativeQueryRequest) evalQueryPredicate(expression schema.Expression) (schema.ExpressionEncoder, error) {
	switch expr := expression.Interface().(type) {
	case *schema.ExpressionAnd:
		exprs := []schema.ExpressionEncoder{}
		for _, nestedExpr := range expr.Expressions {
			evalExpr, err := pr.evalQueryPredicate(nestedExpr)
			if err != nil {
				return nil, err
			}
			if evalExpr != nil {
				exprs = append(exprs, evalExpr)
			}
		}
		return schema.NewExpressionAnd(exprs...), nil
	case *schema.ExpressionBinaryComparisonOperator:
		if expr.Column.Type != schema.ComparisonTargetTypeColumn {
			return nil, fmt.Errorf("%s: unsupported comparison target `%s`", expr.Column.Name, expr.Column.Type)
		}

		switch expr.Column.Name {
		case metadata.TimestampKey:
			switch expr.Operator {
			case metadata.Equal:
				if pr.Timestamp != nil {
					return nil, errors.New("unsupported multiple equality for the timestamp")
				}
				ts, err := pr.getComparisonValue(expr.Value)
				if err != nil {
					return nil, err
				}
				pr.Timestamp = ts
				return nil, nil
			case metadata.Least:
				if pr.End != nil {
					return nil, errors.New("unsupported multiple _lt expressions for the timestamp")
				}
				end, err := pr.getComparisonValue(expr.Value)
				if err != nil {
					return nil, err
				}
				pr.End = end
				return nil, nil
			case metadata.Greater:
				if pr.Start != nil {
					return nil, errors.New("unsupported multiple _gt expressions for the timestamp")
				}
				start, err := pr.getComparisonValue(expr.Value)
				if err != nil {
					return nil, err
				}
				pr.Start = start
				return nil, nil
			default:
				return nil, fmt.Errorf("unsupported operator `%s` for the timestamp", expr.Operator)
			}
		default:
			return expr, nil
		}
	default:
		return nil, fmt.Errorf("unsupported expression: %+v", expression)
	}
}
