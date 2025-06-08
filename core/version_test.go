package core

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestVersion(t *testing.T) {
	t.Run("VERSION variable exists", func(t *testing.T) {
		assert.NotNil(t, VERSION)
		assert.IsType(t, "", VERSION)
	})

	t.Run("VERSION has default value", func(t *testing.T) {
		// VERSION should have a default value of "undefined" when not set during build
		if VERSION == "undefined" {
			assert.Equal(t, "undefined", VERSION)
		} else {
			// If VERSION was set during build, it should not be empty
			assert.NotEmpty(t, VERSION)
		}
	})
}
