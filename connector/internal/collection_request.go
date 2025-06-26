package internal

import (
	"errors"
	"fmt"
	"slices"
	"time"

	"github.com/go-viper/mapstructure/v2"
	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
)

// ColumnOrder the structured sorting columns.
type ColumnOrder struct {
	Name       string
	Descending bool
}

// KeyValue represents a key-value pair.
type KeyValue struct {
	Key   string
	Value any
}

type Grouping struct {
	Dimensions []string
	Aggregates schema.GroupingAggregates
}

// CollectionValidatedArguments hold the common validated arguments.
type CollectionValidatedArguments struct {
	Timestamp *time.Time
	Range     *v1.Range
	OrderBy   []ColumnOrder
	Timeout   time.Duration

	start     *time.Time
	end       *time.Time
	variables map[string]any
	runtime   *metadata.RuntimeSettings
}

// GetStep gets the step duration.
func (cva CollectionValidatedArguments) GetStep() time.Duration {
	if cva.Range == nil {
		return time.Minute
	}

	return cva.Range.Step
}

func (cva CollectionValidatedArguments) getComparisonTimestamp(cmpValue schema.ComparisonValue) (*time.Time, error) {
	rawValue, err := getComparisonValue(cmpValue, cva.variables)
	if err != nil {
		return nil, schema.UnprocessableContentError(err.Error(), map[string]any{
			"field": metadata.TimestampKey,
		})
	}

	return cva.runtime.ParseTimestamp(rawValue)
}

// CollectionRequest the structured predicate result which is evaluated from the raw expression.
type CollectionRequest struct {
	CollectionValidatedArguments

	Value            *schema.ExpressionBinaryComparisonOperator
	LabelExpressions map[string]*LabelExpression
	Functions        []KeyValue
	Groups           *Grouping
}

// EvalCollectionRequest evaluates the requested collection data of the query request.
func EvalCollectionRequest(
	request *schema.QueryRequest,
	arguments map[string]any,
	variables map[string]any,
	runtime *metadata.RuntimeSettings,
) (*CollectionRequest, error) {
	result := &CollectionRequest{
		LabelExpressions: make(map[string]*LabelExpression),
		CollectionValidatedArguments: CollectionValidatedArguments{
			variables: variables,
			runtime:   runtime,
		},
	}

	if len(request.Query.Predicate) > 0 {
		if err := result.evalQueryPredicate(request.Query.Predicate); err != nil {
			return nil, err
		}
	}

	step, err := result.evalArguments(arguments)
	if err != nil {
		return nil, err
	}

	if result.start != nil || result.end != nil {
		result.Range, err = metadata.NewRange(result.start, result.end, step)
		if err != nil {
			return nil, schema.UnprocessableContentError(err.Error(), nil)
		}
	}

	if err := result.evalGroups(request.Query.Groups); err != nil {
		return nil, err
	}

	orderBy, err := evalCollectionOrderBy(request.Query.OrderBy)
	if err != nil {
		return nil, err
	}

	result.OrderBy = orderBy

	return result, nil
}

func (pr *CollectionRequest) evalArguments(arguments map[string]any) (time.Duration, error) {
	if arguments == nil {
		return 0, nil
	}

	if rawTimeout, ok := arguments[metadata.ArgumentKeyTimeout]; ok {
		dur, err := pr.runtime.ParseDuration(rawTimeout)
		if err != nil {
			return 0, fmt.Errorf("failed to parse duration: %w", err)
		}

		pr.Timeout = dur
	}

	var step time.Duration

	if rawStep, ok := arguments[metadata.ArgumentKeyStep]; ok {
		st, err := pr.runtime.ParseDuration(rawStep)
		if err != nil {
			return 0, fmt.Errorf("failed to parse step: %w", err)
		}

		step = st
	}

	fn, ok := arguments[metadata.ArgumentKeyFunctions]
	if !ok || utils.IsNil(fn) {
		return step, nil
	}

	fnMap := []map[string]any{}
	if err := mapstructure.Decode(fn, &fnMap); err != nil {
		return 0, err
	}

	for _, f := range fnMap {
		i := 0

		for k, v := range f {
			if i > 0 {
				return 0, errors.New("each fn item must have 1 function only")
			}

			i++

			pr.Functions = append(pr.Functions, KeyValue{
				Key:   k,
				Value: v,
			})
		}
	}

	return step, nil
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
		return pr.evalExpressionBinaryComparisonOperator(expr)
	default:
		return fmt.Errorf("unsupported expression: %+v", expression)
	}

	return nil
}

func (pr *CollectionRequest) evalExpressionBinaryComparisonOperator(
	expr *schema.ExpressionBinaryComparisonOperator,
) error {
	targetT, err := expr.Column.InterfaceT()
	if err != nil {
		return err
	}

	switch target := targetT.(type) {
	case *schema.ComparisonTargetColumn:
		switch target.Name {
		case metadata.TimestampKey:
			switch expr.Operator {
			case metadata.Equal:
				if pr.Timestamp != nil {
					return errors.New("unsupported multiple equality for the timestamp")
				}

				pr.Timestamp, err = pr.getComparisonTimestamp(expr.Value)
				if err != nil {
					return schema.UnprocessableContentError(err.Error(), map[string]any{
						"field": metadata.TimestampKey,
					})
				}
			case metadata.Least, metadata.LeastOrEqual:
				if pr.end != nil {
					return errors.New("unsupported multiple _lt or _lte expressions for the timestamp")
				}

				pr.end, err = pr.getComparisonTimestamp(expr.Value)
				if err != nil {
					return schema.UnprocessableContentError(err.Error(), map[string]any{
						"field": metadata.TimestampKey,
					})
				}
			case metadata.Greater, metadata.GreaterOrEqual:
				if pr.start != nil {
					return errors.New("unsupported multiple _gt or _gt expressions for the timestamp")
				}

				pr.start, err = pr.getComparisonTimestamp(expr.Value)
				if err != nil {
					return schema.UnprocessableContentError(err.Error(), map[string]any{
						"field": metadata.TimestampKey,
					})
				}
			default:
				return fmt.Errorf("unsupported operator `%s` for the timestamp", expr.Operator)
			}
		case metadata.ValueKey:
			if pr.Value != nil {
				return errors.New("unsupported multiple comparisons for the value")
			}

			pr.Value = expr
		default:
			if le, ok := pr.LabelExpressions[target.Name]; ok {
				le.Expressions = append(le.Expressions, *expr)
			} else {
				pr.LabelExpressions[target.Name] = &LabelExpression{
					Name:        target.Name,
					Expressions: []schema.ExpressionBinaryComparisonOperator{*expr},
				}
			}
		}
	default:
	}

	return nil
}

func (pr *CollectionRequest) evalGroups(grouping *schema.Grouping) error {
	if grouping == nil {
		return nil
	}

	group := Grouping{
		Dimensions: make([]string, len(grouping.Dimensions)),
		Aggregates: grouping.Aggregates,
	}

	for i, dim := range grouping.Dimensions {
		column, err := dim.AsColumn()
		if err != nil {
			return fmt.Errorf("invalid grouping dimension %d: %w", i, err)
		}

		group.Dimensions[i] = column.ColumnName
	}

	pr.Groups = &group

	return nil
}

func evalCollectionOrderBy(orderBy *schema.OrderBy) ([]ColumnOrder, error) {
	var results []ColumnOrder

	if orderBy == nil {
		return results, nil
	}

	for _, elem := range orderBy.Elements {
		switch target := elem.Target.Interface().(type) {
		case *schema.OrderByColumn:
			if slices.Contains([]string{metadata.LabelsKey, metadata.ValuesKey}, target.Name) {
				return nil, fmt.Errorf("ordering by `%s` is unsupported", target.Name)
			}

			orderBy := ColumnOrder{
				Name:       target.Name,
				Descending: elem.OrderDirection == schema.OrderDirectionDesc,
			}
			results = append(results, orderBy)
		default:
			return nil, fmt.Errorf("support ordering by column only, got: %v", elem.Target)
		}
	}

	return results, nil
}
