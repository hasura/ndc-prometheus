package internal

import (
	"errors"
	"fmt"
	"slices"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

// ColumnOrder the structured sorting columns
type ColumnOrder struct {
	Name       string
	Descending bool
}

// KeyValue represents a key-value pair
type KeyValue struct {
	Key   string
	Value any
}

// CollectionRequest the structured predicate result which is evaluated from the raw expression
type CollectionRequest struct {
	Timestamp        schema.ComparisonValue
	Start            schema.ComparisonValue
	End              schema.ComparisonValue
	Value            *schema.ExpressionBinaryComparisonOperator
	LabelExpressions map[string]*LabelExpression
	Functions        []KeyValue
	OrderBy          []ColumnOrder
}

// EvalCollectionRequest evaluates the requested collection data of the query request
func EvalCollectionRequest(request *schema.QueryRequest, arguments map[string]any) (*CollectionRequest, error) {
	result := &CollectionRequest{
		LabelExpressions: make(map[string]*LabelExpression),
	}
	if len(request.Query.Predicate) > 0 {
		if err := result.evalQueryPredicate(request.Query.Predicate); err != nil {
			return nil, err
		}
	}

	if len(arguments) > 0 {
		if fn, ok := arguments[metadata.ArgumentKeyFunctions]; ok && !utils.IsNil(fn) {
			fnMap := []map[string]any{}
			if err := mapstructure.Decode(fn, &fnMap); err != nil {
				return nil, err
			}
			for _, f := range fnMap {
				i := 0
				for k, v := range f {
					if i > 0 {
						return nil, errors.New("each fn item must have 1 function only")
					}
					i++
					result.Functions = append(result.Functions, KeyValue{
						Key:   k,
						Value: v,
					})
				}
			}
		}
	}

	if request.Query.OrderBy != nil {
		for _, elem := range request.Query.OrderBy.Elements {
			switch target := elem.Target.Interface().(type) {
			case *schema.OrderByColumn:
				if slices.Contains([]string{metadata.LabelsKey, metadata.ValuesKey}, target.Name) {
					return nil, fmt.Errorf("ordering by `%s` is unsupported", target.Name)
				}

				orderBy := ColumnOrder{
					Name:       target.Name,
					Descending: elem.OrderDirection == schema.OrderDirectionDesc,
				}
				result.OrderBy = append(result.OrderBy, orderBy)
			default:
				return nil, fmt.Errorf("support ordering by column only, got: %v", elem.Target)
			}
		}
	}
	return result, nil
}

func (pr *CollectionRequest) evalQueryPredicate(expression schema.Expression) error {
	switch expr := expression.Interface().(type) {
	case *schema.ExpressionAnd:
		for _, nestedExpr := range expr.Expressions {
			if err := pr.evalQueryPredicate(nestedExpr); err != nil {
				return err
			}
		}
	case *schema.ExpressionBinaryComparisonOperator:
		if expr.Column.Type != schema.ComparisonTargetTypeColumn {
			return fmt.Errorf("%s: unsupported comparison target `%s`", expr.Column.Name, expr.Column.Type)
		}
		switch expr.Column.Name {
		case metadata.TimestampKey:
			switch expr.Operator {
			case metadata.Equal:
				if pr.Timestamp != nil {
					return errors.New("unsupported multiple equality for the timestamp")
				}
				pr.Timestamp = expr.Value
			case metadata.Least:
				if pr.End != nil {
					return errors.New("unsupported multiple _lt expressions for the timestamp")
				}
				pr.End = expr.Value
			case metadata.Greater:
				if pr.Start != nil {
					return errors.New("unsupported multiple _gt expressions for the timestamp")
				}
				pr.Start = expr.Value
			default:
				return fmt.Errorf("unsupported operator `%s` for the timestamp", expr.Operator)
			}
		case metadata.ValueKey:
			if pr.Value != nil {
				return errors.New("unsupported multiple comparisons for the value")
			}
			pr.Value = expr
		default:
			if le, ok := pr.LabelExpressions[expr.Column.Name]; ok {
				le.Expressions = append(le.Expressions, *expr)
			} else {
				pr.LabelExpressions[expr.Column.Name] = &LabelExpression{
					Name:        expr.Column.Name,
					Expressions: []schema.ExpressionBinaryComparisonOperator{*expr},
				}
			}
		}
	default:
		return fmt.Errorf("unsupported expression: %+v", expression)
	}

	return nil
}
