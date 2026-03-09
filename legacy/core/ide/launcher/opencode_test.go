package launcher

import (
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

func TestOpenCode_implementsLauncher(t *testing.T) {
	var _ Launcher = (*OpenCode)(nil)
}

func TestNewOpenCode(t *testing.T) {
	o := NewOpenCode("/opt/homebrew/bin/opencode")
	assert.Equal(t, "opencode", o.ID())
	assert.Equal(t, "/opt/homebrew/bin/opencode", o.Binary())
}

func TestOpenCode_SupportsSessionID(t *testing.T) {
	o := NewOpenCode("/usr/bin/opencode")
	assert.False(t, o.SupportsSessionID())
}

func TestOpenCode_NewSessionArgs(t *testing.T) {
	o := NewOpenCode("/usr/bin/opencode")
	args := o.NewSessionArgs("", "You are a docs assistant")
	assert.Equal(t, []string{
		"--prompt", "You are a docs assistant",
	}, args)
}

func TestOpenCode_ResumeArgs(t *testing.T) {
	o := NewOpenCode("/usr/bin/opencode")
	args := o.ResumeArgs("ses_abc123", "You are a docs assistant")
	assert.Equal(t, []string{
		"--session", "ses_abc123",
		"--prompt", "You are a docs assistant",
	}, args)
}

func TestParseSessionList(t *testing.T) {
	tests := []struct {
		name    string
		output  string
		wantID  string
		wantErr string
	}{
		{
			name: "valid multi-session output",
			output: "Session ID                      Title                Updated\n" +
				"─────────────────────────────────────────────────────────────\n" +
				"ses_abc123                      My document          12:43 PM\n" +
				"ses_def456                      Another doc          11:00 AM\n",
			wantID: "ses_abc123",
		},
		{
			name: "single session",
			output: "Session ID                      Title                Updated\n" +
				"─────────────────────────────────────────────────────────────\n" +
				"ses_only1                       Only session         3:00 PM\n",
			wantID: "ses_only1",
		},
		{
			name:    "empty output",
			output:  "",
			wantErr: "no opencode sessions found",
		},
		{
			name: "header only",
			output: "Session ID                      Title                Updated\n" +
				"─────────────────────────────────────────────────────────────\n",
			wantErr: "no opencode sessions found",
		},
		{
			name: "no ses_ prefix",
			output: "Session ID                      Title                Updated\n" +
				"─────────────────────────────────────────────────────────────\n" +
				"abc123                          Not a valid ID       12:43 PM\n",
			wantErr: "no opencode sessions found",
		},
		{
			name:   "whitespace around line",
			output: "\n  ses_ws123                     Whitespace test      1:00 PM\n",
			wantID: "ses_ws123",
		},
		{
			name:    "only newlines",
			output:  "\n\n\n",
			wantErr: "no opencode sessions found",
		},
	}

	for _, tt := range tests {
		t.Run(tt.name, func(t *testing.T) {
			id, err := parseSessionList(tt.output)
			if tt.wantErr != "" {
				require.Error(t, err)
				assert.Contains(t, err.Error(), tt.wantErr)
			} else {
				require.NoError(t, err)
				assert.Equal(t, tt.wantID, id)
			}
		})
	}
}
