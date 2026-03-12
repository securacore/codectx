package shared

import (
	"testing"
)

func TestRunWithSpinner_ExecutesAction(t *testing.T) {
	executed := false
	err := RunWithSpinner("test", func() {
		executed = true
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if !executed {
		t.Fatal("action was not executed")
	}
}

func TestRunWithSpinner_NonTTY_NoError(t *testing.T) {
	// In test environments, stdin is not a TTY, so this exercises
	// the non-interactive path — the action runs directly.
	var callOrder []string
	err := RunWithSpinner("step one", func() {
		callOrder = append(callOrder, "first")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	err = RunWithSpinner("step two", func() {
		callOrder = append(callOrder, "second")
	})
	if err != nil {
		t.Fatalf("unexpected error: %v", err)
	}
	if len(callOrder) != 2 || callOrder[0] != "first" || callOrder[1] != "second" {
		t.Fatalf("expected [first, second], got %v", callOrder)
	}
}
