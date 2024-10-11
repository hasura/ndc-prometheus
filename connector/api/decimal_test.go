package api

import (
	"encoding/json"
	"testing"

	"github.com/hasura/ndc-sdk-go/utils"
	"gotest.tools/v3/assert"
)

func TestDecimal(t *testing.T) {
	assert.Assert(t, Decimal{}.IsNil())
	assert.Equal(t, Decimal{}.String(), "Inf")
	assert.Equal(t, Decimal{value: utils.ToPtr(1.2)}.String(), "1.2")

	_, err := NewDecimal("foo")
	assert.ErrorContains(t, err, "failed to convert Float, got: foo")

	dec, err := NewDecimal("1.1")
	assert.NilError(t, err)
	assert.Equal(t, dec.ScalarName(), "Decimal")
	assert.Equal(t, dec.String(), "1.1")

	assert.ErrorContains(t, json.Unmarshal([]byte("foo"), &dec), "invalid character")
	assert.NilError(t, json.Unmarshal([]byte("2.2"), &dec))

	bs, err := json.Marshal(dec)
	assert.NilError(t, err)
	assert.Equal(t, string(bs), `"2.2"`)
}
