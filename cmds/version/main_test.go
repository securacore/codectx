package version_test

import (
	"context"
	"testing"

	"github.com/securacore/codectx/cmds/version"
	"github.com/securacore/codectx/core/project"
)

func TestVersion_DefaultIsDev(t *testing.T) {
	if project.Version != "dev" {
		t.Errorf("expected default version %q, got %q", "dev", project.Version)
	}
}

func TestVersion_CommandName(t *testing.T) {
	if version.Command.Name != "version" {
		t.Errorf("expected command name %q, got %q", "version", version.Command.Name)
	}
}

func TestVersion_CommandUsage(t *testing.T) {
	if version.Command.Usage == "" {
		t.Error("expected non-empty usage string")
	}
}

func TestVersion_ActionReturnsNil(t *testing.T) {
	// The action prints the version and returns nil.
	// We can't easily capture stdout in this test, but we verify no error.
	err := version.Command.Action(context.Background(), version.Command)
	if err != nil {
		t.Errorf("expected nil error, got %v", err)
	}
}

func TestVersion_CanOverride(t *testing.T) {
	original := project.Version
	defer func() { project.Version = original }()

	project.Version = "1.2.3"
	if project.Version != "1.2.3" {
		t.Errorf("expected overridden version %q, got %q", "1.2.3", project.Version)
	}
}
