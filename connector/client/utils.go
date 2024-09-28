package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"time"

	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

const maxSteps = 11000

// UnixTimeUnit the unit for unix timestamp
type UnixTimeUnit string

const (
	UnixTimeSecond      UnixTimeUnit = "s"
	UnixTimeMillisecond UnixTimeUnit = "ms"
)

// Duration returns the duration of the unit
func (ut UnixTimeUnit) Duration() time.Duration {
	switch ut {
	case UnixTimeMillisecond:
		return time.Millisecond
	default:
		return time.Second
	}
}

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
func ParseDuration(value any, unixTimeUnit UnixTimeUnit) (time.Duration, error) {
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
		return time.Duration(reflectValue.Int()) * unixTimeUnit.Duration(), nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		return time.Duration(reflectValue.Uint()) * unixTimeUnit.Duration(), nil
	case reflect.Float32, reflect.Float64:
		return time.Duration(int64(reflectValue.Float() * float64(unixTimeUnit.Duration()))), nil
	default:
		return 0, fmt.Errorf("unable to parse duration from kind %v", kind)
	}
}

// ParseTimestamp parses timestamp from an unknown value
func ParseTimestamp(s any, unixTimeUnit UnixTimeUnit) (*time.Time, error) {
	reflectValue, ok := utils.UnwrapPointerFromReflectValue(reflect.ValueOf(s))
	if !ok {
		return nil, nil
	}

	baseMs := int64(unixTimeUnit.Duration() / time.Millisecond)
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
		// or as a Unix timestamp,
		// with optional decimal places for sub-second precision
		result := time.UnixMilli(reflectValue.Int() * baseMs)
		return &result, nil
	case reflect.Uint, reflect.Uint8, reflect.Uint16, reflect.Uint32, reflect.Uint64:
		result := time.UnixMilli(int64(reflectValue.Uint()) * baseMs)
		return &result, nil
	case reflect.Float32, reflect.Float64:
		result := time.UnixMilli(int64(reflectValue.Float() * float64(baseMs)))
		return &result, nil
	default:
		return nil, fmt.Errorf("unable to parse timestamp from kind %v", kind)
	}
}

type apiResponse struct {
	Status    string          `json:"status"`
	Data      json.RawMessage `json:"data"`
	ErrorType v1.ErrorType    `json:"errorType"`
	Error     string          `json:"error"`
	Warnings  []string        `json:"warnings,omitempty"`
}

func apiError(code int) bool {
	// These are the codes that Prometheus sends when it returns an error.
	return code == http.StatusUnprocessableEntity || code == http.StatusBadRequest
}

func errorTypeAndMsgFor(resp *http.Response) (v1.ErrorType, string) {
	switch resp.StatusCode / 100 {
	case 4:
		return v1.ErrClient, fmt.Sprintf("client error: %d", resp.StatusCode)
	case 5:
		return v1.ErrServer, fmt.Sprintf("server error: %d", resp.StatusCode)
	}
	return v1.ErrBadResponse, fmt.Sprintf("bad response code %d", resp.StatusCode)
}
