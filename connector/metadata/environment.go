package metadata

import (
	"errors"
	"os"
)

// EnvironmentValue represents either a literal string or an environment reference
type EnvironmentValue struct {
	Value    *string `json:"value,omitempty" yaml:"value,omitempty"`
	Variable *string `json:"variable,omitempty" yaml:"variable,omitempty"`
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
		return errors.New("require either value or valueFromEnv")
	}
	if ev.Value != nil && ev.Variable != nil {
		return errors.New("only one of value or valueFromEnv is allowed")
	}
	if ev.Variable != nil && *ev.Variable == "" {
		return errors.New("the variable name of valueFromEnv is empty")
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
