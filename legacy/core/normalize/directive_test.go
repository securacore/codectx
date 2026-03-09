package normalize

import (
	"testing"

	"github.com/stretchr/testify/assert"
)

func TestDirective_notEmpty(t *testing.T) {
	d := Directive()
	assert.NotEmpty(t, d)
	assert.Contains(t, d, "normalization agent")
}

func TestDirective_unrestrictedToolAccess(t *testing.T) {
	d := Directive()
	assert.Contains(t, d, "Use any and all tools available to you")
}

func TestDirective_scopeConstraints(t *testing.T) {
	d := Directive()
	// Must instruct the AI about what NOT to modify.
	assert.Contains(t, d, "docs/packages/")
	assert.Contains(t, d, "code blocks")
	assert.Contains(t, d, "metadata.yml")
}

func TestDirective_phaseStructure(t *testing.T) {
	d := Directive()
	// Must have all 5 phases.
	assert.Contains(t, d, "Phase 1: Read All Documentation")
	assert.Contains(t, d, "Phase 2: Identify Inconsistencies")
	assert.Contains(t, d, "Phase 3: Choose Canonical Forms")
	assert.Contains(t, d, "Phase 4: Apply Normalizations")
	assert.Contains(t, d, "Phase 5: Report")
}

func TestDirective_canonicalFormCriteria(t *testing.T) {
	d := Directive()
	// Must describe how to choose canonical forms.
	assert.Contains(t, d, "Frequency")
	assert.Contains(t, d, "Specificity")
}

func TestDirective_preserveMeaningRule(t *testing.T) {
	d := Directive()
	assert.Contains(t, d, "Preserve meaning exactly")
}

func TestDirective_codeBlockSafetyRule(t *testing.T) {
	d := Directive()
	assert.Contains(t, d, "Never modify code blocks")
}

func TestDirective_yamlFrontMatterSafetyRule(t *testing.T) {
	d := Directive()
	assert.Contains(t, d, "Never modify YAML front matter")
}

func TestAssemblePrompt_alwaysIncludesDocumentationDirectory(t *testing.T) {
	// Documentation Directory section is unconditionally added regardless
	// of whether manifestSummary is empty.
	prompt := AssemblePrompt("any/dir/", "")
	assert.Contains(t, prompt, "## Documentation Directory")
	assert.Contains(t, prompt, "any/dir/")
}

func TestAssemblePrompt_includesDirective(t *testing.T) {
	prompt := AssemblePrompt("docs/", "")
	assert.Contains(t, prompt, "normalization agent")
	assert.Contains(t, prompt, "Documentation Directory")
	assert.Contains(t, prompt, "docs/")
}

func TestAssemblePrompt_includesDocsDir(t *testing.T) {
	prompt := AssemblePrompt("my/custom/docs/", "")
	assert.Contains(t, prompt, "my/custom/docs/")
}

func TestAssemblePrompt_includesManifestSummary(t *testing.T) {
	summary := "### Foundation\n\n- **philosophy**: Guiding principles\n"
	prompt := AssemblePrompt("docs/", summary)
	assert.Contains(t, prompt, "Documentation Map")
	assert.Contains(t, prompt, "philosophy")
}

func TestAssemblePrompt_omitsMapWhenEmpty(t *testing.T) {
	prompt := AssemblePrompt("docs/", "")
	assert.NotContains(t, prompt, "Documentation Map")
}

func TestAssemblePrompt_omitsMapForNoDocumentation(t *testing.T) {
	// BuildManifestSummary returns this when manifest is empty.
	prompt := AssemblePrompt("docs/", "No existing documentation.")
	// The "No existing documentation." string is treated as non-empty,
	// so the map section IS included with that message.
	assert.Contains(t, prompt, "Documentation Map")
}
