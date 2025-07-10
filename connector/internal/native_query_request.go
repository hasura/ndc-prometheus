package internal

import (
	"errors"
	"fmt"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
)

// NativeQueryRequest the structured native request which is evaluated from the raw expression.
type NativeQueryRequest struct {
	CollectionValidatedArguments

	Expression      schema.Expression
	HasValueBoolExp bool
}

// EvalNativeQueryRequest evaluates the requested collection data of the query request.
func EvalNativeQueryRequest(
	request *schema.QueryRequest,
	arguments map[string]any,
	variables map[string]any,
	runtime *metadata.RuntimeSettings,
) (*NativeQueryRequest, error) {
	result := &NativeQueryRequest{
		CollectionValidatedArguments: CollectionValidatedArguments{
			variables: variables,
			runtime:   runtime,
		},
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

func (pr *NativeQueryRequest) evalQueryPredicate(
	expression schema.Expression,
) (schema.ExpressionEncoder, error) {
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
	case *schema.ExpressionOr:
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

		return schema.NewExpressionOr(exprs...), nil
	case *schema.ExpressionNot, *schema.ExpressionUnaryComparisonOperator:
		return expr, nil
	case *schema.ExpressionBinaryComparisonOperator:
		return pr.evalExpressionBinaryComparisonOperator(expr)
	default:
		return nil, fmt.Errorf("unsupported expression: %+v", expression)
	}
}

func (pr *NativeQueryRequest) evalExpressionBinaryComparisonOperator(
	expr *schema.ExpressionBinaryComparisonOperator,
) (schema.ExpressionEncoder, error) {
	targetT, err := expr.Column.InterfaceT()
	if err != nil {
		return nil, err
	}

	switch target := targetT.(type) {
	case *schema.ComparisonTargetColumn:
		return pr.evalExpressionBinaryComparisonOperatorColumn(expr, target)
	default:
		return nil, fmt.Errorf("unsupported comparison target `%s`", targetT.Type())
	}
}

func (pr *NativeQueryRequest) evalExpressionBinaryComparisonOperatorColumn(
	expr *schema.ExpressionBinaryComparisonOperator,
	target *schema.ComparisonTargetColumn,
) (schema.ExpressionEncoder, error) {
	switch target.Name {
	case metadata.TimestampKey:
		switch expr.Operator {
		case metadata.Equal:
			if pr.Timestamp != nil {
				return nil, errors.New("unsupported multiple equality for the timestamp")
			}

			ts, err := pr.getComparisonTimestamp(expr.Value)
			if err != nil {
				return nil, err
			}

			pr.Timestamp = ts

			return nil, nil
		case metadata.Least, metadata.LeastOrEqual:
			if pr.end != nil {
				return nil, errors.New("unsupported multiple _lt expressions for the timestamp")
			}

			end, err := pr.getComparisonTimestamp(expr.Value)
			if err != nil {
				return nil, err
			}

			pr.end = end

			return nil, nil
		case metadata.Greater, metadata.GreaterOrEqual:
			if pr.start != nil {
				return nil, errors.New("unsupported multiple _gt expressions for the timestamp")
			}

			start, err := pr.getComparisonTimestamp(expr.Value)
			if err != nil {
				return nil, err
			}

			pr.start = start

			return nil, nil
		default:
			return nil, fmt.Errorf("unsupported operator `%s` for the timestamp", expr.Operator)
		}
	case metadata.ValueKey:
		pr.HasValueBoolExp = true

		return expr, nil
	default:
		return expr, nil
	}
}
