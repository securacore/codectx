package errs

import (
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestNew_createsError(t *testing.T) {
	err := New(KindNotFound, "resource missing")
	require.NotNil(t, err)
	assert.Equal(t, KindNotFound, err.Kind)
	assert.Equal(t, "resource missing", err.Message)
	assert.Nil(t, err.Err)
}

func TestWrap_createsWrappedError(t *testing.T) {
	cause := fmt.Errorf("connection refused")
	err := Wrap(KindInternal, "fetch failed", cause)
	require.NotNil(t, err)
	assert.Equal(t, KindInternal, err.Kind)
	assert.Equal(t, "fetch failed", err.Message)
	assert.Equal(t, cause, err.Err)
	assert.Equal(t, "fetch failed: connection refused", err.Error())
}
