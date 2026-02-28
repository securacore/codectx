package ide

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirective_notEmpty(t *testing.T) {
	d := Directive()
	assert.NotEmpty(t, d)
	assert.Contains(t, d, "documentation authoring assistant")
}

func TestAssemblePrompt_includesDirective(t *testing.T) {
	prompt := AssemblePrompt("", "", "")
	assert.Contains(t, prompt, "documentation authoring assistant")
	// No extra sections when all are empty.
	assert.NotContains(t, prompt, "Existing Documentation")
	assert.NotContains(t, prompt, "Project Preferences")
	assert.NotContains(t, prompt, "Package Authoring")
}

func TestAssemblePrompt_includesManifest(t *testing.T) {
	prompt := AssemblePrompt("### Foundation\n\n- **philosophy**: Guiding principles\n", "", "")
	assert.Contains(t, prompt, "Existing Documentation")
	assert.Contains(t, prompt, "philosophy")
	assert.NotContains(t, prompt, "Project Preferences")
}

func TestAssemblePrompt_includesPreferences(t *testing.T) {
	prompt := AssemblePrompt("", "- Compression is **enabled**", "")
	assert.Contains(t, prompt, "Project Preferences")
	assert.Contains(t, prompt, "Compression is **enabled**")
}

func TestAssemblePrompt_includesBoth(t *testing.T) {
	prompt := AssemblePrompt("### Topics\n\n- **go**: Go conventions\n", "- Model class: gpt-4o-class", "")
	assert.Contains(t, prompt, "Existing Documentation")
	assert.Contains(t, prompt, "Project Preferences")
	assert.Contains(t, prompt, "go")
	assert.Contains(t, prompt, "gpt-4o-class")
}

func TestAssemblePrompt_includesPackageContext(t *testing.T) {
	pkgCtx := BuildPackageContext(true)
	prompt := AssemblePrompt("", "", pkgCtx)
	assert.Contains(t, prompt, "Package Authoring")
	assert.Contains(t, prompt, "package project")
	assert.Contains(t, prompt, "package/")
}

func TestAssemblePrompt_noPackageContextForRegularProject(t *testing.T) {
	pkgCtx := BuildPackageContext(false)
	prompt := AssemblePrompt("", "", pkgCtx)
	assert.NotContains(t, prompt, "Package Authoring")
}
