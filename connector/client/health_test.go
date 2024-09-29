package client

import (
	"context"
	"testing"

	"gotest.tools/v3/assert"
)

func TestHealth(t *testing.T) {
	c := createTestClient(t)
	assert.Assert(t, c.Healthy(context.TODO()))
}
