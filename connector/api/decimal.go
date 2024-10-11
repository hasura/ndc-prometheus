package api

import (
	"encoding/json"
	"fmt"

	"github.com/hasura/ndc-sdk-go/utils"
)

// Decimal wraps the scalar implementation for big decimal,
// with string representation
//
// @scalar Decimal string
type Decimal struct {
	value *float64
	raw   *string
}

// NewDecimal creates a BigDecimal instance
func NewDecimal[T comparable](value T) (Decimal, error) {
	result := Decimal{}
	if err := result.FromValue(value); err != nil {
		return Decimal{}, err
	}
	return result, nil
}

// ScalarName get the schema name of the scalar
func (bd Decimal) IsNil() bool {
	return bd.raw == nil
}

// Stringer implements fmt.Stringer interface.
func (bd Decimal) String() string {
	if bd.raw != nil {
		return *bd.raw
	}
	if bd.value != nil {
		return fmt.Sprint(*bd.value)
	}
	return "Inf"
}

// MarshalJSON implements json.Marshaler.
func (bi Decimal) MarshalJSON() ([]byte, error) {
	return json.Marshal(bi.String())
}

// UnmarshalJSON implements json.Unmarshaler.
func (bi *Decimal) UnmarshalJSON(b []byte) error {
	var value any
	if err := json.Unmarshal(b, &value); err != nil {
		return fmt.Errorf("invalid decimal value: %s", err)
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
