package errs

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestString(t *testing.T) {
	tests := []struct {
		name     string
		kind     Kind
		expected string
	}{
		{"internal", KindInternal, "internal error"},
		{"not found", KindNotFound, "not found"},
		{"invalid", KindInvalid, "invalid"},
		{"permission", KindPermission, "permission denied"},
		{"conflict", KindConflict, "conflict"},
		{"exists", KindExists, "already exists"},
		{"unknown defaults to internal", Kind(99), "internal error"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, tt.kind.String())
		})
	}
}
