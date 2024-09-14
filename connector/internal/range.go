package internal

import (
	"fmt"
	"strings"

	"github.com/hasura/ndc-sdk-go/utils"
	"github.com/prometheus/common/model"
)

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

// ParseRangeResolution parses the range resolution from a string
func ParseRangeResolution(input any) (*RangeResolution, error) {
	str, err := utils.DecodeNullableString(input)
	if err != nil {
		return nil, fmt.Errorf("invalid range resolution %v: %s", input, err)
	}
	if str == nil {
		return nil, nil
	}
	parts := strings.Split(*str, ":")
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
