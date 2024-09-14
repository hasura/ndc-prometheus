package client

import (
	"fmt"
	"reflect"
	"time"

	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
)

const maxSteps = 11000

// calculate the step to avoid exceeding maximum resolution of 11,000 points per time-series
func evalStepFromRange(start time.Time, end time.Time) time.Duration {
	difference := end.Sub(start)
	switch {
	case difference <= time.Minute:
		return time.Second
	case difference <= time.Hour:
		return time.Minute
	case difference <= 3*time.Hour:
		return 5 * time.Minute
	case difference <= 6*time.Hour:
		return 10 * time.Minute
	case difference <= 12*time.Hour:
		return 30 * time.Minute
	case difference <= 24*time.Hour:
		return time.Hour
	case difference <= maxSteps*24*time.Hour:
		return 24 * time.Hour
	default:
		return 30 * 24 * time.Hour
	}
}

// ParseDuration parses duration from an unknown value
func ParseDuration(value any) (time.Duration, error) {
	reflectValue, ok := utils.UnwrapPointerFromReflectValue(reflect.ValueOf(value))
	if !ok {
		return 0, nil
	}

	kind := reflectValue.Kind()
	switch kind {
	case reflect.Invalid:
		return 0, nil
	case reflect.String:
		strValue := reflectValue.String()
		if d, err := model.ParseDuration(strValue); err == nil {
			return time.Duration(d), nil
		} else {
			return 0, fmt.Errorf("unable to parse duration from string %s: %s", strValue, err)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// or as a number in seconds
		return time.Duration(reflectValue.Int()) * time.Second, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return time.Duration(reflectValue.Uint()) * time.Second, nil
	case reflect.Float32, reflect.Float64:
		floatValue := reflectValue.Float() * 1000

		return time.Duration(int64(floatValue)) * time.Millisecond, nil
	default:
		return 0, fmt.Errorf("unable to parse duration from kind %v", kind)
	}
}

// ParseTimestamp parses timestamp from an unknown value
func ParseTimestamp(s any) (*time.Time, error) {
	reflectValue, ok := utils.UnwrapPointerFromReflectValue(reflect.ValueOf(s))
	if !ok {
		return nil, nil
	}

	kind := reflectValue.Kind()
	switch kind {
	case reflect.Invalid:
		return nil, nil
	case reflect.String:
		strValue := reflectValue.String()
		if strValue == "now" {
			now := time.Now()
			return &now, nil
		}
		// Input timestamps may be provided either in RFC3339 format
		if t, err := time.Parse(time.RFC3339, strValue); err == nil {
			return &t, nil
		}
		if d, err := time.ParseDuration(strValue); err == nil {
			result := time.Now().Add(-d)
			return &result, nil
		} else {
			return nil, fmt.Errorf("unable to parse timestamp from string %s: %s", strValue, err)
		}
	case reflect.Int, reflect.Int8, reflect.Int16, reflect.Int32, reflect.Int64:
		// or as a Unix timestamp in seconds,
		// with optional decimal places for sub-second precision
		result := time.Unix(reflectValue.Int(), 0)
		return &result, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result := time.Unix(int64(reflectValue.Uint()), 0)
		return &result, nil
	case reflect.Float32, reflect.Float64:
		v := int64(reflectValue.Float() * 1000)
		result := time.UnixMilli(v)
		return &result, nil
	default:
		return nil, fmt.Errorf("unable to parse timestamp from kind %v", kind)
	}
}
