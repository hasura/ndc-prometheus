package metadata

import (
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

const (
	Equal          = "_eq"
	NotEqual       = "_neq"
	In             = "_in"
	NotIn          = "_nin"
	Regex          = "_regex"
	NotRegex       = "_nregex"
	Least          = "_lt"
	LeastOrEqual   = "_lte"
	Greater        = "_gt"
	GreaterOrEqual = "_gte"
)

var defaultScalars = map[string]schema.ScalarType{
	string(ScalarBoolean): {
		AggregateFunctions:  schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{},
		Representation:      schema.NewTypeRepresentationBoolean().Encode(),
	},
	string(ScalarString): {
		AggregateFunctions: schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{
			Equal:    schema.NewComparisonOperatorEqual().Encode(),
			In:       schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarJSON))).Encode(),
			NotEqual: schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).Encode(),
			Regex:    schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).Encode(),
			NotRegex: schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarString))).Encode(),
			NotIn:    schema.NewComparisonOperatorCustom(schema.NewArrayType(schema.NewNamedType(string(ScalarString)))).Encode(),
		},
		Representation: schema.NewTypeRepresentationString().Encode(),
	},
	string(ScalarDecimal): {
		AggregateFunctions: schema.ScalarTypeAggregateFunctions{},
		ComparisonOperators: map[string]schema.ComparisonOperatorDefinition{
			Equal:          schema.NewComparisonOperatorEqual().Encode(),
			NotEqual:       schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).Encode(),
			Least:          schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).Encode(),
			LeastOrEqual:   schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).Encode(),
			Greater:        schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).Encode(),
			GreaterOrEqual: schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarDecimal))).Encode(),
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
			Equal:   schema.NewComparisonOperatorEqual().Encode(),
			Least:   schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarTimestamp))).Encode(),
			Greater: schema.NewComparisonOperatorCustom(schema.NewNamedType(string(ScalarTimestamp))).Encode(),
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
	},
	objectName_QueryResultValues: {
		Description: utils.ToPtr("A general query result with values and labels"),
		Fields:      createQueryResultValuesObjectFields(),
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
	},
	objectName_HoltWintersInput: {
		Description: utils.ToPtr("Input arguments for the holt_winters function"),
		Fields: schema.ObjectTypeFields{
			"sf": schema.ObjectField{
				Description: utils.ToPtr("The lower the smoothing factor sf, the more importance is given to old data. Must be between 0 and 1"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"tf": schema.ObjectField{
				Description: utils.ToPtr("The higher the trend factor tf, the more trends in the data is considered. Must be between 0 and 1"),
				Type:        schema.NewNamedType(string(ScalarFloat64)).Encode(),
			},
			"range": schema.ObjectField{
				Description: utils.ToPtr("The range value"),
				Type:        schema.NewNamedType(string(ScalarDuration)).Encode(),
			},
		},
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
		Description: utils.ToPtr("Evaluation timestamp. Use this argument if you want to run an instant query"),
		Type:        schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyTimeout: {
		Description: utils.ToPtr("Evaluation timeout"),
		Type:        schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyStart: {
		Description: utils.ToPtr("Start timestamp. Use this argument if you want to run an range query"),
		Type:        schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyEnd: {
		Description: utils.ToPtr("End timestamp. Use this argument if you want to run an range query"),
		Type:        schema.NewNullableNamedType(string(ScalarTimestamp)).Encode(),
	},
	ArgumentKeyStep: {
		Description: utils.ToPtr("Query resolution step width in duration format or float number of seconds"),
		Type:        schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyOffset: {
		Description: utils.ToPtr("The offset modifier allows changing the time offset for individual instant and range vectors in a query"),
		Type:        schema.NewNullableNamedType(string(ScalarDuration)).Encode(),
	},
	ArgumentKeyFlat: {
		Description: utils.ToPtr("Flatten grouped values out the root array"),
		Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
	},
}
