package main

import (
	"testing"

	"gotest.tools/v3/assert"
)

func TestJsonSchemaConfiguration(t *testing.T) {
	assert.NilError(t, jsonSchemaConfiguration())
}
