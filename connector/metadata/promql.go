package metadata

import (
	"fmt"

	"github.com/hasura/ndc-sdk-go/schema"
	"github.com/hasura/ndc-sdk-go/utils"
)

func buildLabelJoinObjectTypeName(name string) string {
	return fmt.Sprintf("%sLabelJoinInput", name)
}

func buildLabelReplaceObjectTypeName(name string) string {
	return fmt.Sprintf("%sLabelReplaceInput", name)
}

func createLabelJoinObjectType(labelEnumScalarName string) schema.ObjectType {
	return schema.ObjectType{
		Description: utils.ToPtr("Input arguments for the label_join function"),
		Fields: schema.ObjectTypeFields{
			"dest_label": schema.ObjectField{
				Description: utils.ToPtr("The destination label name"),
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			},
			"separator": schema.ObjectField{
				Description: utils.ToPtr("The separator between source labels"),
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			},
			"source_labels": schema.ObjectField{
				Description: utils.ToPtr("Source labels"),
				Type:        schema.NewArrayType(schema.NewNamedType(labelEnumScalarName)).Encode(),
			},
		},
	}
}

func createLabelReplaceObjectType(labelEnumScalarName string) schema.ObjectType {
	return schema.ObjectType{
		Description: utils.ToPtr("Input arguments for the label_replace function"),
		Fields: schema.ObjectTypeFields{
			"dest_label": schema.ObjectField{
				Description: utils.ToPtr("The destination label name"),
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			},
			"replacement": schema.ObjectField{
				Description: utils.ToPtr("The replacement value"),
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			},
			"source_label": schema.ObjectField{
				Description: utils.ToPtr("Source label"),
				Type:        schema.NewNamedType(labelEnumScalarName).Encode(),
			},
			"regex": schema.ObjectField{
				Description: utils.ToPtr("The regular expression against the value of the source label"),
				Type:        schema.NewNamedType(string(ScalarString)).Encode(),
			},
		},
	}
}

func createPromQLFunctionObjectFields(name string, labelEnumScalarName string) schema.ObjectTypeFields {
	typeArrayLabels := schema.NewNullableType(schema.NewArrayType(schema.NewNamedType(labelEnumScalarName))).Encode()

	return schema.ObjectTypeFields{
		string(TopK): schema.ObjectField{
			Description: utils.ToPtr("Largest k elements by sample value"),
			Type:        schema.NewNullableNamedType(string(ScalarInt64)).Encode(),
		},
		string(BottomK): schema.ObjectField{
			Description: utils.ToPtr("Smallest k elements by sample value"),
			Type:        schema.NewNullableNamedType(string(ScalarInt64)).Encode(),
		},
		string(LimitK): schema.ObjectField{
			Description: utils.ToPtr("Limit sample n elements"),
			Type:        schema.NewNullableNamedType(string(ScalarInt64)).Encode(),
		},
		string(Quantile): schema.ObjectField{
			Description: utils.ToPtr("Calculate φ-quantile (0 ≤ φ ≤ 1) over dimensions"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(LimitRatio): schema.ObjectField{
			Description: utils.ToPtr("Sample elements with approximately 𝑟 ratio if 𝑟 > 0, and the complement of such samples if 𝑟 = -(1.0 - 𝑟))"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(Absolute): schema.ObjectField{
			Description: utils.ToPtr("Returns the input vector with all sample values converted to their absolute value"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Absent): schema.ObjectField{
			Description: utils.ToPtr("Returns an empty vector if the vector passed to it has any elements (floats or native histograms) and a 1-element vector with the value 1 if the vector passed to it has no elements"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(AbsentOverTime): schema.ObjectField{
			Description: utils.ToPtr("Returns an empty vector if the range vector passed to it has any elements (floats or native histograms) and a 1-element vector with the value 1 if the range vector passed to it has no elements"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Ceil): schema.ObjectField{
			Description: utils.ToPtr("Rounds the sample values of all elements in v up to the nearest integer value greater than or equal to v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Changes): schema.ObjectField{
			Description: utils.ToPtr("Returns the number of times its value has changed within the provided time range as an instant vector"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Clamp): schema.ObjectField{
			Description: utils.ToPtr("Clamps the sample values of all elements in v to have a lower limit of min and an upper limit of max"),
			Type:        schema.NewNullableNamedType(string(objectName_ValueBoundaryInput)).Encode(),
		},
		string(ClampMax): schema.ObjectField{
			Description: utils.ToPtr("Clamps the sample values of all elements in v to have an upper limit of max"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(ClampMin): schema.ObjectField{
			Description: utils.ToPtr("Clamps the sample values of all elements in v to have a lower limit of min"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(Delta): schema.ObjectField{
			Description: utils.ToPtr("Calculates the difference between the first and last value of each time series element in a range vector v, returning an instant vector with the given deltas and equivalent labels"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Derivative): schema.ObjectField{
			Description: utils.ToPtr("Calculates the per-second derivative of the time series in a range vector v, using simple linear regression"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Exponential): schema.ObjectField{
			Description: utils.ToPtr("Calculates the exponential function for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Floor): schema.ObjectField{
			Description: utils.ToPtr("Rounds the sample values of all elements in v down to the nearest integer value smaller than or equal to v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HistogramAvg): schema.ObjectField{
			Description: utils.ToPtr("Returns the arithmetic average of observed values stored in a native histogram. Samples that are not native histograms are ignored and do not show up in the returned vector"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HistogramCount): schema.ObjectField{
			Description: utils.ToPtr("Returns the count of observations stored in a native histogram. Samples that are not native histograms are ignored and do not show up in the returned vector"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HistogramSum): schema.ObjectField{
			Description: utils.ToPtr("Returns the sum of observations stored in a native histogram"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HistogramFraction): schema.ObjectField{
			Description: utils.ToPtr("Returns the estimated fraction of observations between the provided lower and upper values. Samples that are not native histograms are ignored and do not show up in the returned vector"),
			Type:        schema.NewNullableNamedType(string(objectName_ValueBoundaryInput)).Encode(),
		},
		string(HistogramQuantile): schema.ObjectField{
			Description: utils.ToPtr("Calculates the φ-quantile (0 ≤ φ ≤ 1) from a classic histogram or from a native histogram"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(HistogramStddev): schema.ObjectField{
			Description: utils.ToPtr("Returns the estimated standard deviation of observations in a native histogram, based on the geometric mean of the buckets where the observations lie. Samples that are not native histograms are ignored and do not show up in the returned vector"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HistogramStdvar): schema.ObjectField{
			Description: utils.ToPtr("Returns the estimated standard variance of observations in a native histogram"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(HoltWinters): schema.ObjectField{
			Description: utils.ToPtr("Produces a smoothed value for time series based on the range in v"),
			Type:        schema.NewNullableNamedType(string(objectName_HoltWintersInput)).Encode(),
		},
		string(IDelta): schema.ObjectField{
			Description: utils.ToPtr("Calculates the difference between the last two samples in the range vector v, returning an instant vector with the given deltas and equivalent labels"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Increase): schema.ObjectField{
			Description: utils.ToPtr("Calculates the increase in the time series in the range vector. Breaks in monotonicity (such as counter resets due to target restarts) are automatically adjusted for"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(IRate): schema.ObjectField{
			Description: utils.ToPtr("Calculates the per-second instant rate of increase of the time series in the range vector. This is based on the last two data points"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(LabelJoin): schema.ObjectField{
			Description: utils.ToPtr("Joins all the values of all the src_labels using separator and returns the timeseries with the label dst_label containing the joined value"),
			Type:        schema.NewNullableNamedType(buildLabelJoinObjectTypeName(name)).Encode(),
		},
		string(LabelReplace): schema.ObjectField{
			Description: utils.ToPtr("Matches the regular expression regex against the value of the label src_label. If it matches, the value of the label dst_label in the returned timeseries will be the expansion of replacement, together with the original labels in the input"),
			Type:        schema.NewNullableNamedType(buildLabelReplaceObjectTypeName(name)).Encode(),
		},
		string(Ln): schema.ObjectField{
			Description: utils.ToPtr("Calculates the natural logarithm for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Log2): schema.ObjectField{
			Description: utils.ToPtr("Calculates the binary logarithm for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Log10): schema.ObjectField{
			Description: utils.ToPtr("Calculates the decimal logarithm for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(PredictLinear): schema.ObjectField{
			Description: utils.ToPtr("Predicts the value of time series t seconds from now, based on the range vector v, using simple linear regression"),
			Type:        schema.NewNullableNamedType(objectName_PredictLinearInput).Encode(),
		},
		string(Rate): schema.ObjectField{
			Description: utils.ToPtr("Calculates the per-second average rate of increase of the time series in the range vector"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Resets): schema.ObjectField{
			Description: utils.ToPtr("Returns the number of counter resets within the provided time range as an instant vector"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Round): schema.ObjectField{
			Description: utils.ToPtr("Rounds the sample values of all elements in v to the nearest integer"),
			Type:        schema.NewNullableNamedType(string(ScalarFloat64)).Encode(),
		},
		string(Scalar): schema.ObjectField{
			Description: utils.ToPtr("Returns the sample value of that single element as a scalar"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Sgn): schema.ObjectField{
			Description: utils.ToPtr("Returns a vector with all sample values converted to their sign, defined as this: 1 if v is positive, -1 if v is negative and 0 if v is equal to zero"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Sort): schema.ObjectField{
			Description: utils.ToPtr("Returns vector elements sorted by their sample values, in ascending order. Native histograms are sorted by their sum of observations"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(SortDesc): schema.ObjectField{
			Description: utils.ToPtr("Same as sort, but sorts in descending order"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(SortByLabel): schema.ObjectField{
			Description: utils.ToPtr("Returns vector elements sorted by their label values and sample value in case of label values being equal, in ascending order"),
			Type:        typeArrayLabels,
		},
		string(SortByLabelDesc): schema.ObjectField{
			Description: utils.ToPtr("Same as sort_by_label, but sorts in descending order"),
			Type:        typeArrayLabels,
		},
		string(Sqrt): schema.ObjectField{
			Description: utils.ToPtr("Calculates the square root of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Timestamp): schema.ObjectField{
			Description: utils.ToPtr("Returns the timestamp of each of the samples of the given vector as the number of seconds since January 1, 1970 UTC. It also works with histogram samples"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(AvgOverTime): schema.ObjectField{
			Description: utils.ToPtr("The average value of all points in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(MinOverTime): schema.ObjectField{
			Description: utils.ToPtr("The minimum value of all points in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(MaxOverTime): schema.ObjectField{
			Description: utils.ToPtr("The maximum value of all points in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(SumOverTime): schema.ObjectField{
			Description: utils.ToPtr("The sum of all values in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(CountOverTime): schema.ObjectField{
			Description: utils.ToPtr("The count of all values in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(QuantileOverTime): schema.ObjectField{
			Description: utils.ToPtr("The φ-quantile (0 ≤ φ ≤ 1) of the values in the specified interval"),
			Type:        schema.NewNullableNamedType(objectName_QuantileOverTimeInput).Encode(),
		},
		string(StddevOverTime): schema.ObjectField{
			Description: utils.ToPtr("The population standard deviation of the values in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(StdvarOverTime): schema.ObjectField{
			Description: utils.ToPtr("The population standard variance of the values in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(LastOverTime): schema.ObjectField{
			Description: utils.ToPtr("The most recent point value in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(PresentOverTime): schema.ObjectField{
			Description: utils.ToPtr("The value 1 for any series in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(MadOverTime): schema.ObjectField{
			Description: utils.ToPtr("The median absolute deviation of all points in the specified interval"),
			Type:        schema.NewNullableNamedType(string(ScalarRangeResolution)).Encode(),
		},
		string(Acos): schema.ObjectField{
			Description: utils.ToPtr("Calculates the arccosine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Acosh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the inverse hyperbolic cosine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Asin): schema.ObjectField{
			Description: utils.ToPtr("Calculates the arcsine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Asinh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the inverse hyperbolic sine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Atan): schema.ObjectField{
			Description: utils.ToPtr("Calculates the arctangent of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Atanh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the inverse hyperbolic tangent of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Cos): schema.ObjectField{
			Description: utils.ToPtr("Calculates the cosine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Cosh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the hyperbolic cosine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Sin): schema.ObjectField{
			Description: utils.ToPtr("Calculates the sine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Sinh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the hyperbolic sine of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Tan): schema.ObjectField{
			Description: utils.ToPtr("Calculates the tangent of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Tanh): schema.ObjectField{
			Description: utils.ToPtr("Calculates the hyperbolic tangent of all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Deg): schema.ObjectField{
			Description: utils.ToPtr("Converts radians to degrees for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
		string(Rad): schema.ObjectField{
			Description: utils.ToPtr("Converts degrees to radians for all elements in v"),
			Type:        schema.NewNullableNamedType(string(ScalarBoolean)).Encode(),
		},
	}
}
