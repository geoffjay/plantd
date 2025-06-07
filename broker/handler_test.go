package main

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

type TestCallback struct {
	HandlerCallback
}

// TestNewHandler tests the NewHandler function.
func TestNewHandler(t *testing.T) {
	h := NewHandler()

	assert.NotNil(t, h)
	assert.Equal(t, h.running, false)
	assert.Equal(t, h.canceled, false)
	assert.Equal(t, len(h.callbacks), 0)
}

// TestAddCallback tests the AddCallback function.
func TestAddCallback(t *testing.T) {
	h := NewHandler()

	assert.NotNil(t, h)

	h.AddCallback("test", &TestCallback{})

	assert.Equal(t, len(h.callbacks), 1)
}

// TestGetCallback tests the GetCallback function.
func TestGetCallback(t *testing.T) {
	h := NewHandler()

	assert.NotNil(t, h)

	h.AddCallback("test", &TestCallback{})

	assert.Equal(t, len(h.callbacks), 1)

	callback, err := h.GetCallback("test")

	assert.Nil(t, err)
	assert.NotNil(t, callback)

	_, err = h.GetCallback("test2")

	assert.ErrorContains(t, err, "callback not found for test2")
}
