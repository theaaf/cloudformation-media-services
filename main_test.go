package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

type TestEmbed struct {
	Foo int
	Bar bool
}

func TestReshapeProps(t *testing.T) {
	in := map[string]interface{}{
		"A": []interface{}{
			map[string]interface{}{
				"B": "123",
			},
		},
		"Foo": 1,
		"Bar": "true",
	}

	var out struct {
		TestEmbed
		A []*struct {
			B *int
		}
	}

	ReshapeProps(in, &out)

	require.Len(t, out.A, 1)
	require.NotNil(t, out.A[0])
	require.NotNil(t, out.A[0].B)
	assert.Equal(t, 123, *out.A[0].B)
	assert.Equal(t, 1, out.Foo)
	assert.True(t, out.Bar)
}
