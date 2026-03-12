package shared

import (
	"testing"

	"github.com/securacore/codectx/core/project"
)

func TestShouldAutoCompile_SkipFlag(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(true)}
	got := ShouldAutoCompile(prefsCfg, false, true, "recompile")
	if got {
		t.Error("expected false when --no-compile is set")
	}
}

func TestShouldAutoCompile_ForceFlag(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(false)}
	got := ShouldAutoCompile(prefsCfg, true, false, "recompile")
	if !got {
		t.Error("expected true when --compile is set, even with auto_compile: false")
	}
}

func TestShouldAutoCompile_ConfigDefault(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: nil}
	got := ShouldAutoCompile(prefsCfg, false, false, "recompile")
	if !got {
		t.Error("expected true when auto_compile is not set (default)")
	}
}

func TestShouldAutoCompile_ConfigExplicitTrue(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(true)}
	got := ShouldAutoCompile(prefsCfg, false, false, "recompile")
	if !got {
		t.Error("expected true when auto_compile is explicitly true")
	}
}

func TestShouldAutoCompile_ConfigExplicitFalse(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(false)}
	got := ShouldAutoCompile(prefsCfg, false, false, "recompile")
	if got {
		t.Error("expected false when auto_compile is explicitly false")
	}
}

func TestShouldAutoCompile_SkipOverridesForce(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(true)}
	got := ShouldAutoCompile(prefsCfg, true, true, "recompile")
	if got {
		t.Error("expected false when --no-compile is set, even with --compile")
	}
}

func TestShouldAutoCompile_DifferentActionLabel(t *testing.T) {
	t.Parallel()

	prefsCfg := &project.PreferencesConfig{AutoCompile: project.BoolPtr(false)}
	got := ShouldAutoCompile(prefsCfg, false, false, "initial compile")
	if got {
		t.Error("expected false with auto_compile: false regardless of action label")
	}
}
