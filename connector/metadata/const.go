package metadata

import (
	"fmt"
	"slices"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

type ScalarName string

const (
	ScalarBoolean   ScalarName = "Boolean"
	ScalarInt64     ScalarName = "Int64"
	ScalarFloat64   ScalarName = "Float64"
	ScalarString    ScalarName = "String"
	ScalarDecimal   ScalarName = "Decimal"
	ScalarTimestamp ScalarName = "Timestamp"
	ScalarLabelSet  ScalarName = "LabelSet"
	ScalarDuration  ScalarName = "Duration"
	ScalarJSON      ScalarName = "JSON"
)

const (
	FunctionPromQLQuery = "promql_query"
)

// LabelComparisonOperator represents a label comparison operator enum.
type LabelComparisonOperator string

const (
	LabelEqual                    LabelComparisonOperator = "_eq"
	LabelNotEqual                 LabelComparisonOperator = "_neq"
	LabelIn                       LabelComparisonOperator = "_in"
	LabelNotIn                    LabelComparisonOperator = "_nin"
	LabelRegex                    LabelComparisonOperator = "_regex"
	LabelNotRegex                 LabelComparisonOperator = "_nregex"
	LabelContains                 LabelComparisonOperator = "_contains"
	LabelNotContains              LabelComparisonOperator = "_ncontains"
	LabelContainsInsensitive      LabelComparisonOperator = "_icontains"
	LabelNotContainsInsensitive   LabelComparisonOperator = "_nicontains"
	LabelStartsWith               LabelComparisonOperator = "_starts_with"
	LabelNotStartsWith            LabelComparisonOperator = "_nstarts_with"
	LabelStartsWithInsensitive    LabelComparisonOperator = "_istarts_with"
	LabelNotStartsWithInsensitive LabelComparisonOperator = "_nistarts_with"
	LabelEndsWith                 LabelComparisonOperator = "_ends_with"
	LabelNotEndsWith              LabelComparisonOperator = "_nends_with"
	LabelEndsWithInsensitive      LabelComparisonOperator = "_iends_with"
	LabelNotEndsWithInsensitive   LabelComparisonOperator = "_niends_with"
)

var enumValuesNegativeLabelComparisonOperator = []LabelComparisonOperator{
	LabelNotEqual,
	LabelNotIn,
	LabelNotRegex,
	LabelNotContains,
	LabelNotContainsInsensitive,
	LabelNotStartsWith,
	LabelNotStartsWithInsensitive,
	LabelNotEndsWith,
	LabelNotEndsWithInsensitive,
}

var enumValuesPositiveLabelComparisonOperator = []LabelComparisonOperator{
	LabelEqual,
	LabelIn,
	LabelRegex,
	LabelContains,
	LabelContainsInsensitive,
	LabelStartsWith,
	LabelStartsWithInsensitive,
	LabelEndsWith,
	LabelEndsWithInsensitive,
}

var enumValuesLabelComparisonOperator = append(enumValuesPositiveLabelComparisonOperator, enumValuesNegativeLabelComparisonOperator...)

// ParseLabelComparisonOperator parses the label comparison operator from string.
func ParseLabelComparisonOperator(input string) (LabelComparisonOperator, error) {
	result := LabelComparisonOperator(input)

	if !slices.Contains(enumValuesLabelComparisonOperator, result) {
		return "", fmt.Errorf("invalid label comparison operator; expected one of %v, got %s", enumValuesLabelComparisonOperator, input)
	}

	return result, nil
}

// IsNegative checks if the current operator is negative.
func (lco LabelComparisonOperator) IsNegative() bool {
	return slices.Contains(enumValuesPositiveLabelComparisonOperator, lco)
}

// Negation returns the negative operator of the current one.
func (lco LabelComparisonOperator) Negation() LabelComparisonOperator {
	switch lco {
	case LabelEqual:
		return LabelNotEqual
	case LabelNotEqual:
		return LabelEqual
	case LabelIn:
		return LabelNotIn
	case LabelNotIn:
		return LabelIn
	case LabelRegex:
		return LabelNotRegex
	case LabelNotRegex:
		return LabelRegex
	case LabelContains:
		return LabelNotContains
	case LabelNotContains:
		return LabelContains
	case LabelStartsWith:
		return LabelNotStartsWith
	case LabelNotStartsWith:
		return LabelStartsWith
	case LabelStartsWithInsensitive:
		return LabelNotStartsWithInsensitive
	case LabelNotStartsWithInsensitive:
		return LabelStartsWithInsensitive
	case LabelEndsWith:
		return LabelNotEndsWith
	case LabelNotEndsWith:
		return LabelEndsWith
	case LabelEndsWithInsensitive:
		return LabelNotEndsWithInsensitive
	case LabelNotEndsWithInsensitive:
		return LabelEndsWithInsensitive
	default:
		return ""
	}
}

// ComparisonOperator represents a value comparison operator enum.
type ComparisonOperator string

const (
	Equal          ComparisonOperator = "_eq"
	NotEqual       ComparisonOperator = "_neq"
	Less           ComparisonOperator = "_lt"
	LessOrEqual    ComparisonOperator = "_lte"
	Greater        ComparisonOperator = "_gt"
	GreaterOrEqual ComparisonOperator = "_gte"
)

var enumValuesComparisonOperator = []ComparisonOperator{
	Equal,
	NotEqual,
	Less,
	LessOrEqual,
	Greater,
	GreaterOrEqual,
}

// ParseComparisonOperator parses the comparison operator from string.
func ParseComparisonOperator(input string) (ComparisonOperator, error) {
	result := ComparisonOperator(input)

	if !slices.Contains(enumValuesComparisonOperator, result) {
		return "", fmt.Errorf("invalid comparison operator; expected one of %v, got %s", enumValuesComparisonOperator, input)
	}

	return result, nil
}

// Negation returns the negative operator of the current one.
func (lco ComparisonOperator) Negation() ComparisonOperator {
	switch lco {
	case Equal:
		return NotEqual
	case NotEqual:
		return Equal
	case Greater:
		return Less
	case GreaterOrEqual:
		return LessOrEqual
	case Less:
		return Greater
	case LessOrEqual:
		return GreaterOrEqual
	default:
		return ""
	}
}

var defaultScalars = map[string]schema.ScalarType{
	string(ScalarBoolean): {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationBoolean().Encode(),
	},
	string(ScalarString): {
		AggregateFunctions: schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{
			string(LabelEqual): schema.NewComparisonOperatorEqual().Encode(),
			string(LabelIn): schema.NewComparisonOperatorIn().
				Encode(),
			string(LabelNotEqual): schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).
				Encode(),
			string(LabelRegex): schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).
				Encode(),
			string(LabelNotRegex): schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).
				Encode(),
			string(LabelNotIn): schema.NewComparisonOperatorCustom(schema.NewArrayType(schema.NewNamedType(string(ScalarString)))).
				Encode(),
			string(LabelContains):              schema.NewComparisonOperatorContains().Encode(),
			string(LabelContainsInsensitive):   schema.NewComparisonOperatorContainsInsensitive().Encode(),
			string(LabelStartsWith):            schema.NewComparisonOperatorStartsWith().Encode(),
			string(LabelStartsWithInsensitive): schema.NewComparisonOperatorStartsWithInsensitive().Encode(),
			string(LabelEndsWith):              schema.NewComparisonOperatorEndsWith().Encode(),
			string(LabelEndsWithInsensitive):   schema.NewComparisonOperatorEndsWithInsensitive().Encode(),
		},
		Representation: schema.NewTypeRepresentationString().Encode(),
	},
	string(ScalarDecimal): {
		AggregateFunctions: schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{
			string(Equal): schema.NewComparisonOperatorEqual().Encode(),
			string(NotEqual): schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).
				Encode(),
			string(Less): schema.NewComparisonOperatorLessThan().
				Encode(),
			string(LessOrEqual): schema.NewComparisonOperatorLessThanOrEqual().Encode(),
			string(Greater): schema.NewComparisonOperatorGreaterThan().
				Encode(),
			string(GreaterOrEqual): schema.NewComparisonOperatorGreaterThanOrEqual().
				Encode(),
		},
		Representation: schema.NewTypeRepresentationBigDecimal().Encode(),
	},
	string(ScalarDuration): {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationJSON().Encode(),
	},
	string(ScalarTimestamp): {
		AggregateFunctions: schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{
			string(Equal): schema.NewComparisonOperatorEqual().Encode(),
			string(NotEqual): schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).
				Encode(),
			string(Less): schema.NewComparisonOperatorLessThan().
				Encode(),
			string(LessOrEqual): schema.NewComparisonOperatorLessThanOrEqual().Encode(),
			string(Greater): schema.NewComparisonOperatorGreaterThan().
				Encode(),
			string(GreaterOrEqual): schema.NewComparisonOperatorGreaterThanOrEqual().
				Encode(),
		},
		Representation: schema.NewTypeRepresentationTimestamp().Encode(),
	},
	string(ScalarLabelSet): {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationJSON().Encode(),
	},
}

const (
	TimestampKey = "timestamp"
	ValueKey     = "value"
	ValuesKey    = "values"
	LabelsKey    = "labels"
)

type PromQLFunctionName string

const (
	Sum               PromQLFunctionName = "sum"
	Min               PromQLFunctionName = "min"
	Max               PromQLFunctionName = "max"
	Avg               PromQLFunctionName = "avg"
	Count             PromQLFunctionName = "count"
	CountValues       PromQLFunctionName = "count_values"
	Stddev            PromQLFunctionName = "stddev"
	Stdvar            PromQLFunctionName = "stdvar"
	TopK              PromQLFunctionName = "topk"
	BottomK           PromQLFunctionName = "bottomk"
	Quantile          PromQLFunctionName = "quantile"
	LimitK            PromQLFunctionName = "limitk"
	LimitRatio        PromQLFunctionName = "limit_ratio"
	Group             PromQLFunctionName = "group"
	Absolute          PromQLFunctionName = "abs"
	Absent            PromQLFunctionName = "absent"
	AbsentOverTime    PromQLFunctionName = "absent_over_time"
	Ceil              PromQLFunctionName = "ceil"
	Changes           PromQLFunctionName = "changes"
	Clamp             PromQLFunctionName = "clamp"
	ClampMax          PromQLFunctionName = "clamp_max"
	ClampMin          PromQLFunctionName = "clamp_min"
	DayOfMonth        PromQLFunctionName = "day_of_month"
	DayOfWeek         PromQLFunctionName = "day_of_week"
	DayOfYear         PromQLFunctionName = "day_of_year"
	DaysInMonth       PromQLFunctionName = "days_in_month"
	Delta             PromQLFunctionName = "delta"
	Derivative        PromQLFunctionName = "deriv"
	Exponential       PromQLFunctionName = "exp"
	Floor             PromQLFunctionName = "floor"
	HistogramAvg      PromQLFunctionName = "histogram_avg"
	HistogramCount    PromQLFunctionName = "histogram_count"
	HistogramSum      PromQLFunctionName = "histogram_sum"
	HistogramFraction PromQLFunctionName = "histogram_fraction"
	HistogramQuantile PromQLFunctionName = "histogram_quantile"
	HistogramStddev   PromQLFunctionName = "histogram_stddev"
	HistogramStdvar   PromQLFunctionName = "histogram_stdvar"
	HoltWinters       PromQLFunctionName = "holt_winters"
	Hour              PromQLFunctionName = "hour"
	IDelta            PromQLFunctionName = "idelta"
	Increase          PromQLFunctionName = "increase"
	IRate             PromQLFunctionName = "irate"
	LabelJoin         PromQLFunctionName = "label_join"
	LabelReplace      PromQLFunctionName = "label_replace"
	Ln                PromQLFunctionName = "ln"
	Log2              PromQLFunctionName = "log2"
	Log10             PromQLFunctionName = "log10"
	Minute            PromQLFunctionName = "minute"
	Month             PromQLFunctionName = "month"
	PredictLinear     PromQLFunctionName = "predict_linear"
	Rate              PromQLFunctionName = "rate"
	Resets            PromQLFunctionName = "resets"
	Round             PromQLFunctionName = "round"
	Scalar            PromQLFunctionName = "scalar"
	Sgn               PromQLFunctionName = "sgn"
	Sort              PromQLFunctionName = "sort"
	SortDesc          PromQLFunctionName = "sort_desc"
	SortByLabel       PromQLFunctionName = "sort_by_label"
	SortByLabelDesc   PromQLFunctionName = "sort_by_label_desc"
	Sqrt              PromQLFunctionName = "sqrt"
	Time              PromQLFunctionName = "time"
	Timestamp         PromQLFunctionName = "timestamp"
	Year              PromQLFunctionName = "year"
	AvgOverTime       PromQLFunctionName = "avg_over_time"
	MinOverTime       PromQLFunctionName = "min_over_time"
	MaxOverTime       PromQLFunctionName = "max_over_time"
	SumOverTime       PromQLFunctionName = "sum_over_time"
	CountOverTime     PromQLFunctionName = "count_over_time"
	QuantileOverTime  PromQLFunctionName = "quantile_over_time"
	StddevOverTime    PromQLFunctionName = "stddev_over_time"
	StdvarOverTime    PromQLFunctionName = "stdvar_over_time"
	LastOverTime      PromQLFunctionName = "last_over_time"
	PresentOverTime   PromQLFunctionName = "present_over_time"
	MadOverTime       PromQLFunctionName = "mad_over_time"
	Acos              PromQLFunctionName = "acos"
	Acosh             PromQLFunctionName = "acosh"
	Asin              PromQLFunctionName = "asin"
	Asinh             PromQLFunctionName = "asinh"
	Atan              PromQLFunctionName = "atan"
	Atanh             PromQLFunctionName = "atanh"
	Cos               PromQLFunctionName = "cos"
	Cosh              PromQLFunctionName = "cosh"
	Sin               PromQLFunctionName = "sin"
	Sinh              PromQLFunctionName = "sinh"
	Tan               PromQLFunctionName = "tan"
	Tanh              PromQLFunctionName = "tanh"
	Deg               PromQLFunctionName = "deg"
	Rad               PromQLFunctionName = "rad"
)

var enumValuesPromQLFunctionName = []PromQLFunctionName{
	Sum,
	Min,
	Max,
	Avg,
	Count,
	CountValues,
	Stddev,
	Stdvar,
	TopK,
	BottomK,
	Quantile,
	LimitK,
	LimitRatio,
	Group,
	Absolute,
	Absent,
	AbsentOverTime,
	Ceil,
	Changes,
	Clamp,
	ClampMax,
	ClampMin,
	DayOfMonth,
	DayOfWeek,
	DayOfYear,
	DaysInMonth,
	Delta,
	Derivative,
	Exponential,
	Floor,
	HistogramAvg,
	HistogramCount,
	HistogramSum,
	HistogramFraction,
	HistogramQuantile,
	HistogramStddev,
	HistogramStdvar,
	HoltWinters,
	Hour,
	IDelta,
	Increase,
	IRate,
	LabelJoin,
	LabelReplace,
	Ln,
	Log2,
	Log10,
	Minute,
	Month,
	PredictLinear,
	Rate,
	Resets,
	Round,
	Scalar,
	Sgn,
	Sort,
	SortDesc,
	SortByLabel,
	SortByLabelDesc,
	Sqrt,
	Time,
	Timestamp,
	Year,
	AvgOverTime,
	MinOverTime,
	MaxOverTime,
	SumOverTime,
	CountOverTime,
	QuantileOverTime,
	StddevOverTime,
	StdvarOverTime,
	LastOverTime,
	PresentOverTime,
	MadOverTime,
	Acos,
	Acosh,
	Asin,
	Asinh,
	Atan,
	Atanh,
	Cos,
	Cosh,
	Sin,
	Sinh,
	Tan,
	Tanh,
	Deg,
	Rad,
}

// ParsePromQLFunctionName parses the promQL function name.
func ParsePromQLFunctionName(input string) (PromQLFunctionName, error) {
	result := PromQLFunctionName(input)

	if !slices.Contains(enumValuesPromQLFunctionName, result) {
		return "", fmt.Errorf("invalid PromQLFunctionName; expected one of %v; got %s", enumValuesPromQLFunctionName, input)
	}

	return result, nil
}

const (
	objectName_QueryResultValue           = "QueryResultValue"
	objectName_QueryResultValueWithLabels = "QueryResultValueWithLabels"
	objectName_QueryResultValues          = "QueryResultValues"
	objectName_ValueBoundaryInput         = "ValueBoundaryInput"
	objectName_HoltWintersInput           = "HoltWintersInput"
	objectName_PredictLinearInput         = "PredictLinearInput"
	objectName_QuantileOverTimeInput      = "QuantileOverTimeInput"
)

var defaultObjectTypes = map[string]schema.ObjectType{
	objectName_QueryResultValue: {
		Description: utils.ToPtr("A value of the query result"),
		Fields:      createQueryResultValueObjectFields(),
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
	objectName_QueryResultValues: {
		Description: utils.ToPtr("A general query result with values and labels"),
		Fields:      createQueryResultValuesObjectFields(),
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
	objectName_ValueBoundaryInput: {
		Description: utils.ToPtr("Boundary input arguments"),
		Fields: schema.ObjectTypeFields{
			"min": schema.ObjectField{
				Description: utils.ToPtr("The lower limit of values"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"max": schema.ObjectField{
				Description: utils.ToPtr("The upper limit of values"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
		},
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
	objectName_HoltWintersInput: {
		Description: utils.ToPtr("Input arguments for the holt_winters function"),
		Fields: schema.ObjectTypeFields{
			"sf": schema.ObjectField{
				Description: utils.ToPtr(
					"The lower the smoothing factor sf, the more importance is given to old data. Must be between 0 and 1",
				),
				Type: schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"tf": schema.ObjectField{
				Description: utils.ToPtr(
					"The higher the trend factor tf, the more trends in the data is considered. Must be between 0 and 1",
				),
				Type: schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"range": schema.ObjectField{
				Description: utils.ToPtr("The range value"),
				Type:        schema.NewNamedType(string(ScalarDuration)).Encode(),
			},
		},
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
	objectName_PredictLinearInput: {
		Description: utils.ToPtr("Input arguments for the predict_linear function"),
		Fields: schema.ObjectTypeFields{
			"t": schema.ObjectField{
				Description: utils.ToPtr("Number of seconds from now"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"range": schema.ObjectField{
				Description: utils.ToPtr("The range value"),
				Type:        schema.NewNamedType(string(ScalarDuration)).Encode(),
			},
		},
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
	objectName_QuantileOverTimeInput: {
		Description: utils.ToPtr("Input arguments for the quantile_over_time function"),
		Fields: schema.ObjectTypeFields{
			"quantile": schema.ObjectField{
				Description: utils.ToPtr("The φ-quantile (0 ≤ φ ≤ 1) of the values"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"range": schema.ObjectField{
				Description: utils.ToPtr("The range value"),
				Type:        schema.NewNamedType(string(ScalarDuration)).Encode(),
			},
		},
		ForeignKeys: schema.ObjectTypeForeignKeys{},
	},
}

const (
	ArgumentKeyFlat      = "flat"
	ArgumentKeyTime      = "time"
	ArgumentKeyTimeout   = "timeout"
	ArgumentKeyStart     = "start"
	ArgumentKeyEnd       = "end"
	ArgumentKeyStep      = "step"
	ArgumentKeyOffset    = "offset"
	ArgumentKeyQuery     = "query"
	ArgumentKeyFunctions = "fn"
)

var defaultArgumentInfos = map[string]schema.ArgumentInfo{
	ArgumentKeyTime: {
		Description: utils.ToPtr(
			"Evaluation timestamp. Use this argument if you want to run an instant query",
		),
		Type: schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyTimeout: {
		Description: utils.ToPtr("Evaluation timeout"),
		Type:        schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyStart: {
		Description: utils.ToPtr(
			"Start timestamp. Use this argument if you want to run an range query",
		),
		Type: schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyEnd: {
		Description: utils.ToPtr(
			"End timestamp. Use this argument if you want to run an range query",
		),
		Type: schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyStep: {
		Description: utils.ToPtr(
			"Query resolution step width in duration format or float number of seconds",
		),
		Type: schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyOffset: {
		Description: utils.ToPtr(
			"The offset modifier allows changing the time offset for individual instant and range vectors in a query",
		),
		Type: schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyFlat: {
		Description: utils.ToPtr("Flatten grouped values out the root array"),
		Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
	},
}
