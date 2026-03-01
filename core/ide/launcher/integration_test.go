//go:build integration

package launcher

import (
	"bytes"
	"os/exec"
	"testing"

	coreide "github.com/securacore/codectx/core/ide"
	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// TestIntegration_Claude_directiveWorks runs a quick, non-interactive Claude CLI
// call with the assembled documentation authoring directive and validates the AI
// understands its role. Uses --system-prompt (full replacement) to isolate the
// directive's effect — in production we use --append-system-prompt.
//
// Run: go test -tags integration -run TestIntegration_Claude ./core/ide/launcher/
func TestIntegration_Claude_directiveWorks(t *testing.T) {
	path, err := exec.LookPath("claude")
	if err != nil {
		t.Skip("claude binary not found on PATH")
	}

	directive := coreide.AssemblePrompt("", "", "")

	cmd := exec.Command(path,
		"-p", "In one sentence, what is your role and purpose?",
		"--system-prompt", directive,
	)

	var out bytes.Buffer
	cmd.Stdout = &out
	cmd.Stderr = &bytes.Buffer{}

	err = cmd.Run()
	require.NoError(t, err, "claude CLI should exit cleanly")

	response := out.String()
	t.Logf("Claude response: %s", response)

	// The response should reference documentation authoring in some form.
	lower := bytes.ToLower([]byte(response))
	assert.True(t,
		bytes.Contains(lower, []byte("documentation")) ||
			bytes.Contains(lower, []byte("codectx")) ||
			bytes.Contains(lower, []byte("authoring")) ||
			bytes.Contains(lower, []byte("document")),
		"expected response to mention documentation/authoring, got: %s", response,
	)
}
