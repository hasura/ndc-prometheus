package types

import (
	"errors"
	"os"
)

var (
	errEnvironmentValueRequired    = errors.New("require either value or valueFromEnv")
	errEnvironmentVariableRequired = errors.New("the variable name of valueFromEnv is empty")
	errEnvironmentEitherValueOrEnv = errors.New("only one of value or valueFromEnv is allowed")
)

// EnvironmentValue represents either a literal string or an environment reference
type EnvironmentValue struct {
	Value    *string `json:"value,omitempty" yaml:"value,omitempty" jsonschema:"oneof_required=value"`
	Variable *string `json:"env,omitempty" yaml:"env,omitempty" jsonschema:"oneof_required=env"`
}

// NewEnvironmentValue create an EnvironmentValue with a literal value
func NewEnvironmentValue(value string) EnvironmentValue {
	return EnvironmentValue{
		Value: &value,
	}
}

// NewEnvironmentVariable create an EnvironmentValue with a variable name
func NewEnvironmentVariable(name string) EnvironmentValue {
	return EnvironmentValue{
		Variable: &name,
	}
}

// Validate checks if the current instance is valid
func (ev EnvironmentValue) Validate() error {
	if ev.Value == nil && ev.Variable == nil {
		return errEnvironmentValueRequired
	}
	if ev.Value != nil && ev.Variable != nil {
		return errEnvironmentEitherValueOrEnv
	}
	if ev.Variable != nil && *ev.Variable == "" {
		return errEnvironmentVariableRequired
	}

	return nil
}

// Get gets literal value or from system environment
func (ev EnvironmentValue) Get() (string, error) {
	if err := ev.Validate(); err != nil {
		return "", err
	}
	if ev.Value != nil {
		return *ev.Value, nil
	}

	return os.Getenv(*ev.Variable), nil
}
