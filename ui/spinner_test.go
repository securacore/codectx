package ui

import (
	"errors"
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestSpinErr_success(t *testing.T) {
	called := false
	err := SpinErr("Working...", func() error {
		called = true
		return nil
	})
	assert.NoError(t, err)
	assert.True(t, called)
}

func TestSpinErr_returnsError(t *testing.T) {
	expected := errors.New("something failed")
	err := SpinErr("Working...", func() error {
		return expected
	})
	assert.Equal(t, expected, err)
}

func TestSpin_runsAction(t *testing.T) {
	called := false
	Spin("Working...", func() {
		called = true
	})
	assert.True(t, called)
}
