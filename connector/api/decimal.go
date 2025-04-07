package api

import (
	"encoding/json"
	"fmt"
	"math"

	"github.com/hasura/ndc-sdk-go/utils"
)

// @scalar Decimal string.
type Decimal struct {
	value *float64
	raw   *string
}

// NewDecimal creates a Decimal instance.
func NewDecimal[T comparable](value T) (Decimal, error) {
	result := Decimal{}

	if err := result.FromValue(value); err != nil {
		return Decimal{}, err
	}

	return result, nil
}

// NewDecimalValue creates a Decimal instance from a number value.
func NewDecimalValue[T int | int8 | int16 | int32 | int64 | uint | uint8 | uint16 | uint32 | uint64 | float32 | float64](
	value T,
) Decimal {
	v := float64(value)

	return Decimal{value: &v}
}

// ScalarName get the schema name of the scalar.
func (bd Decimal) IsNil() bool {
	return bd.raw == nil
}

// Value returns the decimal value.
func (bd Decimal) Value() any {
	if bd.value == nil {
		if bd.raw != nil {
			return *bd.raw
		}

		return nil
	}

	if math.IsNaN(*bd.value) {
		return "NaN"
	}

	if *bd.value > 0 && math.IsInf(*bd.value, 1) {
		return "+Inf"
	}

	if *bd.value < 0 && math.IsInf(*bd.value, -1) {
		return "-Inf"
	}

	return *bd.value
}

// Stringer implements fmt.Stringer interface.
func (bd Decimal) String() string {
	v := bd.Value()
	if v == nil {
		return "NaN"
	}

	return fmt.Sprint(v)
}

// MarshalJSON implements json.Marshaler.
func (bi Decimal) MarshalJSON() ([]byte, error) {
	v := bi.Value()
	if v != nil {
		v = fmt.Sprint(v)
	}

	return json.Marshal(v)
}

// UnmarshalJSON implements json.Unmarshaler.
func (bi *Decimal) UnmarshalJSON(b []byte) error {
	var value any

	if err := json.Unmarshal(b, &value); err != nil {
		return fmt.Errorf("invalid decimal value: %w", err)
	}

	return bi.FromValue(value)
}

// FromValue decode any value to Int64.
func (bi *Decimal) FromValue(value any) error {
	fValue, err := utils.DecodeNullableFloat[float64](value)
	if err != nil {
		return err
	}

	if fValue != nil {
		bi.value = fValue
		rawValue := fmt.Sprint(value)
		bi.raw = &rawValue
	}

	return nil
}
