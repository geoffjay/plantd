package util

import (
	"os"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestGetenv(t *testing.T) {
	assert.Equal(t, Getenv("TEST", "default"), "default")
	assert.Equal(t, Getenv("TEST", ""), "")
	_ = os.Setenv("TEST", "test")
	assert.Equal(t, Getenv("TEST", "default"), "test")
}
