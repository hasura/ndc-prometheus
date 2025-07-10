package metadata

import (
	"fmt"
	"math"
	"reflect"
	"strings"
	"time"

	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// UnixTimeUnit the unit for unix timestamp.
type UnixTimeUnit string

const (
	UnixTimeSecond UnixTimeUnit = "s"
	UnixTimeMilli  UnixTimeUnit = "ms"
	UnixTimeMicro  UnixTimeUnit = "us"
	UnixTimeNano   UnixTimeUnit = "ns"
)

// Duration returns the duration of the unit.
func (ut UnixTimeUnit) Duration() time.Duration {
	switch ut {
	case UnixTimeMilli:
		return time.Millisecond
	case UnixTimeMicro:
		return time.Microsecond
	case UnixTimeNano:
		return time.Nanosecond
	default:
		return time.Second
	}
}

// NewRange creates the time range.
func NewRange(start *time.Time, end *time.Time, step time.Duration) (*v1.Range, error) {
	result := v1.Range{
		End:  time.Now(),
		Step: step,
	}

	if start != nil {
		result.Start = *start
	}

	if end != nil {
		result.End = *end
	}

	if result.Step == 0 {
		result.Step = evalStepFromRange(result.Start, result.End)
	}

	return &result, nil
}

// calculate the step to avoid exceeding maximum resolution of 11,000 points per time-series.
func evalStepFromRange(start time.Time, end time.Time) time.Duration {
	difference := end.Sub(start)

	switch {
	case difference <= 5*time.Minute:
		return time.Second
	case difference <= time.Hour:
		return 15 * time.Second
	case difference <= 2*time.Hour:
		return 30 * time.Second
	case difference <= 4*time.Hour:
		return time.Minute
	case difference <= 12*time.Hour:
		return 2 * time.Minute
	case difference <= 24*time.Hour:
		return 5 * time.Minute
	case difference <= 48*time.Hour:
		return 10 * time.Minute
	case difference <= 4*24*time.Hour:
		return 20 * time.Minute
	case difference <= 7*24*time.Hour:
		return 30 * time.Minute
	case difference <= 30*24*time.Hour:
		return time.Hour
	case difference <= 30*30*24*time.Hour:
		base := 30 * 24 * time.Hour
		stepDays := math.Ceil(float64(difference) / float64(base))

		return time.Duration(stepDays) * base
	default:
		return 365 * 24 * time.Hour // year
	}
}

// RangeResolution represents the given range and resolution with format xx:xx.
type RangeResolution struct {
	Range      model.Duration
	Resolution model.Duration
}

// String implements the fmt.Stringer interface.
func (rr RangeResolution) String() string {
	if rr.Resolution == 0 {
		return rr.Range.String()
	}

	return fmt.Sprintf("%s:%s", rr.Range.String(), rr.Resolution.String())
}

// ParseDuration parses duration from an unknown value.
func ParseDuration(value any, unixTimeUnit UnixTimeUnit) (time.Duration, error) {
	result, err := utils.DecodeNullableDuration(value, utils.WithBaseUnix(unixTimeUnit.Duration()))
	if err != nil {
		return 0, err
	}

	if result == nil {
		return 0, nil
	}

	return *result, nil
}

// ParseRangeResolution parses the range resolution from a string.
func ParseRangeResolution(input any, unixTimeUnit UnixTimeUnit) (*RangeResolution, error) {
	reflectValue, ok := utils.UnwrapPointerFromReflectValue(reflect.ValueOf(input))
	if !ok {
		return nil, nil
	}

	kind := reflectValue.Kind()

	if kind != reflect.String {
		rng, err := utils.DecodeDuration(
			reflectValue.Interface(),
			utils.WithBaseUnix(unixTimeUnit.Duration()),
		)
		if err != nil {
			return nil, fmt.Errorf("invalid range resolution %v: %w", input, err)
		}

		return &RangeResolution{Range: model.Duration(rng)}, nil
	}

	parts := strings.Split(reflectValue.String(), ":")
	if parts[0] == "" {
		return nil, fmt.Errorf("invalid range resolution %v", input)
	}

	rng, err := model.ParseDuration(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid duration %s: %w", parts[0], err)
	}

	result := &RangeResolution{
		Range: rng,
	}

	if len(parts) > 1 {
		resolution, err := model.ParseDuration(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid resolution %s: %w", parts[1], err)
		}

		result.Resolution = resolution
	}

	return result, nil
}
