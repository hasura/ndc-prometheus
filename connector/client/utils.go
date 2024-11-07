package client

import (
	"encoding/json"
	"fmt"
	"net/http"
	"reflect"
	"strings"
	"time"

	"github.com/hasura/ndc-sdk-go/utils"
	v1 "github.com/prometheus/client_golang/api/prometheus/v1"
	"github.com/prometheus/common/model"
)

// UnixTimeUnit the unit for unix timestamp
type UnixTimeUnit string

const (
	UnixTimeSecond UnixTimeUnit = "s"
	UnixTimeMilli  UnixTimeUnit = "ms"
	UnixTimeMicro  UnixTimeUnit = "us"
	UnixTimeNano   UnixTimeUnit = "ns"
)

// Duration returns the duration of the unit
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

// calculate the step to avoid exceeding maximum resolution of 11,000 points per time-series
func evalStepFromRange(start time.Time, end time.Time) time.Duration {
	difference := end.Sub(start)
	switch {
	case difference <= time.Minute:
		return time.Second
	case difference <= time.Hour:
		return time.Minute
	default:
		return difference / 60
	}
}

// RangeResolution represents the given range and resolution with format xx:xx
type RangeResolution struct {
	Range      model.Duration
	Resolution model.Duration
}

// String implements the fmt.Stringer interface
func (rr RangeResolution) String() string {
	if rr.Resolution == 0 {
		return rr.Range.String()
	}
	return fmt.Sprintf("%s:%s", rr.Range.String(), rr.Resolution.String())
}

// ParseDuration parses duration from an unknown value
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

// ParseRangeResolution parses the range resolution from a string
func ParseRangeResolution(input any, unixTimeUnit UnixTimeUnit) (*RangeResolution, error) {
	reflectValue, ok := utils.UnwrapPointerFromReflectValue(reflect.ValueOf(input))
	if !ok {
		return nil, nil
	}

	kind := reflectValue.Kind()
	if kind != reflect.String {
		rng, err := utils.DecodeDuration(reflectValue.Interface(), utils.WithBaseUnix(unixTimeUnit.Duration()))
		if err != nil {
			return nil, fmt.Errorf("invalid range resolution %v: %s", input, err)
		}
		return &RangeResolution{Range: model.Duration(rng)}, nil
	}

	parts := strings.Split(reflectValue.String(), ":")
	if parts[0] == "" {
		return nil, fmt.Errorf("invalid range resolution %v", input)
	}

	rng, err := model.ParseDuration(parts[0])
	if err != nil {
		return nil, fmt.Errorf("invalid duration %s: %s", parts[0], err)
	}

	result := &RangeResolution{
		Range: rng,
	}
	if len(parts) > 1 {
		resolution, err := model.ParseDuration(parts[1])
		if err != nil {
			return nil, fmt.Errorf("invalid resolution %s: %s", parts[1], err)
		}
		result.Resolution = resolution
	}
	return result, nil
}

// ParseTimestamp parses timestamp from an unknown value
func ParseTimestamp(s any, unixTimeUnit UnixTimeUnit) (*time.Time, error) {
	return utils.DecodeNullableDateTime(s, utils.WithBaseUnix(unixTimeUnit.Duration()))
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
