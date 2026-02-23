package resolve

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestInferSource(t *testing.T) {
	tests := []struct {
		name     string
		pkgName  string
		author   string
		expected string
	}{
		{"standard", "react", "facebook", "https://github.com/facebook/codectx-react.git"},
		{"hyphenated name", "my-lib", "org", "https://github.com/org/codectx-my-lib.git"},
		{"single char", "x", "y", "https://github.com/y/codectx-x.git"},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			assert.Equal(t, tt.expected, InferSource(tt.pkgName, tt.author))
		})
	}
}
