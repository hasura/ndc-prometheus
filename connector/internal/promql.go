package internal

import (
	"errors"
	"fmt"
	"strings"

	"github.com/hasura/ndc-prometheus/connector/metadata"
	"github.com/hasura/ndc-sdk-go/utils"
)

// ValueBoundaryInput represents the lower and upper input arguments.
type ValueBoundaryInput struct {
	Min float64 `mapstructure:"min"`
	Max float64 `mapstructure:"max"`
}

// LabelJoinInput represents input arguments for the replace_label function.
type LabelJoinInput struct {
	DestLabel    string   `mapstructure:"dest_label"`
	Separator    string   `mapstructure:"separator"`
	SourceLabels []string `mapstructure:"source_labels"`
}

// String implements the fmt.Stringer interface.
func (lji LabelJoinInput) String() string {
	return fmt.Sprintf(
		`"%s", "%s"%s`,
		lji.DestLabel,
		lji.Separator,
		buildPromQLParametersFromStringSlice(lji.SourceLabels),
	)
}

// LabelReplaceInput represents input arguments for the replace_label function.
type LabelReplaceInput struct {
	DestLabel   string `mapstructure:"dest_label"`
	Replacement string `mapstructure:"replacement"`
	SourceLabel string `mapstructure:"source_label"`
	Regex       string `mapstructure:"regex"`
}

// String implements the fmt.Stringer interface.
func (lri LabelReplaceInput) String() string {
	return fmt.Sprintf(
		`"%s", "%s", "%s", "%s"`,
		lri.DestLabel,
		lri.Replacement,
		lri.SourceLabel,
		lri.Regex,
	)
}

// HoltWintersInput represents input arguments of the holt_winters function.
type HoltWintersInput struct {
	Sf    float64
	Tf    float64
	Range metadata.RangeResolution
}

// FromValue decodes data from any value.
func (hwi *HoltWintersInput) FromValue(value any, unixTimeUnit metadata.UnixTimeUnit) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid HoltWintersInput value, expected map, got: %v", value)
	}

	sf, err := utils.GetFloat[float64](m, "sf")
	if err != nil {
		return fmt.Errorf("invalid HoltWintersInput smoothing factor: %w", err)
	}

	if sf <= 0 || sf >= 1 {
		return fmt.Errorf(
			"invalid HoltWintersInput smoothing factor. Expected: 0 < sf < 1, got: %f",
			sf,
		)
	}

	tf, err := utils.GetFloat[float64](m, "tf")
	if err != nil {
		return fmt.Errorf("invalid HoltWintersInput tf: %w", err)
	}

	if tf <= 0 || tf >= 1 {
		return fmt.Errorf(
			"invalid HoltWintersInput trend factor. Expected: 0 < tf < 1, got: %f",
			sf,
		)
	}

	rng, err := metadata.ParseRangeResolution(m["range"], unixTimeUnit)
	if err != nil {
		return fmt.Errorf("invalid HoltWintersInput range: %w", err)
	}

	if rng == nil {
		return errors.New("the range property of HoltWintersInput is required")
	}

	hwi.Sf = sf
	hwi.Tf = tf
	hwi.Range = *rng

	return nil
}

// PredictLinearInput represents input arguments of the predict_linear function.
type PredictLinearInput struct {
	T     float64
	Range metadata.RangeResolution
}

// FromValue decodes data from any value.
func (pli *PredictLinearInput) FromValue(value any, unixTimeUnit metadata.UnixTimeUnit) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid PredictLinearInput value, expected map, got: %v", value)
	}

	t, err := utils.GetFloat[float64](m, "t")
	if err != nil {
		return fmt.Errorf("invalid PredictLinearInput t: %w", err)
	}

	rng, err := metadata.ParseRangeResolution(m["range"], unixTimeUnit)
	if err != nil {
		return fmt.Errorf("invalid PredictLinearInput range: %w", err)
	}

	if rng == nil {
		return errors.New("the range property of PredictLinearInput is required")
	}

	pli.T = t
	pli.Range = *rng

	return nil
}

// QuantileOverTimeInput represents input arguments of the quantile_over_time function.
type QuantileOverTimeInput struct {
	Quantile float64
	Range    metadata.RangeResolution
}

// FromValue decodes data from any value.
func (qoti *QuantileOverTimeInput) FromValue(value any, unixTimeUnit metadata.UnixTimeUnit) error {
	m, ok := value.(map[string]any)
	if !ok {
		return fmt.Errorf("invalid PredictLinearInput value, expected map, got: %v", value)
	}

	quantile, err := utils.GetFloat[float64](m, "quantile")
	if err != nil {
		return fmt.Errorf("invalid PredictLinearInput quantile: %w", err)
	}

	if quantile < 0 || quantile > 1 {
		return fmt.Errorf("quantile value should be between 0 and 1, got %f", quantile)
	}

	rng, err := metadata.ParseRangeResolution(m["range"], unixTimeUnit)
	if err != nil {
		return fmt.Errorf("invalid PredictLinearInput range: %w", err)
	}

	if rng == nil {
		return errors.New("the range property of PredictLinearInput is required")
	}

	qoti.Quantile = quantile
	qoti.Range = *rng

	return nil
}

func buildPromQLParametersFromStringSlice(inputs []string) string {
	if len(inputs) == 0 {
		return ""
	}

	builder := strings.Builder{}
	for _, str := range inputs {
		_, _ = builder.WriteString(`, "`)
		_, _ = builder.WriteString(str)
		_, _ = builder.WriteRune('"')
	}

	return builder.String()
}
