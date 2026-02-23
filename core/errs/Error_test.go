package errs

import (
	"errors"
	"fmt"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestError_messageOnly(t *testing.T) {
	err := &Error{Kind: KindNotFound, Message: "file missing"}
	assert.Equal(t, "file missing", err.Error())
}

func TestError_withWrappedError(t *testing.T) {
	inner := fmt.Errorf("disk full")
	err := &Error{Kind: KindInternal, Message: "write failed", Err: inner}
	assert.Equal(t, "write failed: disk full", err.Error())
}

func TestUnwrap_returnsWrappedError(t *testing.T) {
	inner := fmt.Errorf("underlying cause")
	err := &Error{Kind: KindInternal, Message: "wrapper", Err: inner}
	assert.Equal(t, inner, err.Unwrap())
}

func TestUnwrap_returnsNilWhenNoWrapped(t *testing.T) {
	err := &Error{Kind: KindInternal, Message: "no cause"}
	assert.Nil(t, err.Unwrap())
}

func TestIs_matchesSameKind(t *testing.T) {
	err1 := &Error{Kind: KindNotFound, Message: "a"}
	err2 := &Error{Kind: KindNotFound, Message: "b"}
	assert.True(t, err1.Is(err2))
}

func TestIs_differentKind(t *testing.T) {
	err1 := &Error{Kind: KindNotFound, Message: "a"}
	err2 := &Error{Kind: KindConflict, Message: "b"}
	assert.False(t, err1.Is(err2))
}

func TestIs_nonErrorTarget(t *testing.T) {
	err := &Error{Kind: KindNotFound, Message: "a"}
	assert.False(t, err.Is(fmt.Errorf("plain error")))
}

func TestIs_worksWithErrorsIs(t *testing.T) {
	inner := &Error{Kind: KindInvalid, Message: "bad input"}
	wrapped := fmt.Errorf("context: %w", inner)
	target := &Error{Kind: KindInvalid}
	assert.True(t, errors.Is(wrapped, target))
}
