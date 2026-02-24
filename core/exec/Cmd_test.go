package exec

import (
	"bytes"
	"os"
	osexec "os/exec"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
)

// --- Cmd ---

func TestCmd_singleWord(t *testing.T) {
	cmd := Cmd("echo")
	// cmd.Path is resolved to an absolute path by exec.Command;
	// Args[0] retains the original value.
	assert.Equal(t, []string{"echo"}, cmd.Args)
}

func TestCmd_multipleArgs(t *testing.T) {
	cmd := Cmd("git commit -m message")
	assert.Equal(t, []string{"git", "commit", "-m", "message"}, cmd.Args)
}

func TestCmd_singleQuotedSpaces(t *testing.T) {
	cmd := Cmd("echo 'hello world'")
	assert.Equal(t, []string{"echo", "'hello world'"}, cmd.Args)
}

func TestCmd_doubleQuotedSpaces(t *testing.T) {
	cmd := Cmd(`echo "hello world"`)
	assert.Equal(t, []string{"echo", `"hello world"`}, cmd.Args)
}

func TestCmd_backtickQuoted(t *testing.T) {
	cmd := Cmd("echo `hello world`")
	assert.Equal(t, []string{"echo", "`hello world`"}, cmd.Args)
}

func TestCmd_multipleSpaces(t *testing.T) {
	cmd := Cmd("git   commit    -m   message")
	assert.Equal(t, []string{"git", "commit", "-m", "message"}, cmd.Args)
}

func TestCmd_withOptions(t *testing.T) {
	cmd := Cmd("ls -la", Dir("/tmp"))
	assert.Equal(t, []string{"ls", "-la"}, cmd.Args)
	assert.Equal(t, "/tmp", cmd.Dir)
}

func TestCmd_emptyQuotes(t *testing.T) {
	// Empty single quotes produce an empty token: the quotes themselves.
	cmd := Cmd("echo ''")
	assert.Equal(t, []string{"echo", "''"}, cmd.Args)
}

func TestCmd_mixedQuotes(t *testing.T) {
	cmd := Cmd(`git commit -m "it's working"`)
	assert.Equal(t, []string{"git", "commit", "-m", `"it's working"`}, cmd.Args)
}

func TestCmd_trailingSpaces(t *testing.T) {
	cmd := Cmd("echo hello   ")
	assert.Equal(t, []string{"echo", "hello"}, cmd.Args)
}

func TestCmd_leadingSpaces(t *testing.T) {
	cmd := Cmd("  echo hello")
	assert.Equal(t, []string{"echo", "hello"}, cmd.Args)
}

// --- Dir ---

func TestDir_setsDir(t *testing.T) {
	cmd := Cmd("echo test", Dir("/some/path"))
	assert.Equal(t, "/some/path", cmd.Dir)
}

// --- Stdin ---

func TestStdin_setsStdin(t *testing.T) {
	cmd := Cmd("cat")
	Stdin(cmd)
	assert.Equal(t, os.Stdin, cmd.Stdin)
}

// --- Stdout ---

func TestStdout_setsStdout(t *testing.T) {
	cmd := Cmd("echo test")
	Stdout(cmd)
	assert.Equal(t, os.Stdout, cmd.Stdout)
}

// --- Stderr ---

func TestStderr_setsStderr(t *testing.T) {
	cmd := Cmd("echo test")
	Stderr(cmd)
	assert.Equal(t, os.Stderr, cmd.Stderr)
}

// --- Stdio ---

func TestStdio_setsAllStreams(t *testing.T) {
	cmd := Cmd("echo test")
	Stdio(cmd)
	assert.Equal(t, os.Stdin, cmd.Stdin)
	assert.Equal(t, os.Stdout, cmd.Stdout)
	assert.Equal(t, os.Stderr, cmd.Stderr)
}

// --- Cmd with custom option ---

func TestCmd_customOption(t *testing.T) {
	var buf bytes.Buffer
	cmd := Cmd("echo test", func(c *osexec.Cmd) {
		c.Stdout = &buf
	})
	require.NotNil(t, cmd)
	assert.Equal(t, &buf, cmd.Stdout)
}

func TestCmd_emptyStringPanics(t *testing.T) {
	assert.Panics(t, func() {
		Cmd("")
	})
}

func TestCmd_whitespaceOnlyPanics(t *testing.T) {
	assert.Panics(t, func() {
		Cmd("   ")
	})
}

func TestCmd_multipleOptions(t *testing.T) {
	var buf bytes.Buffer
	cmd := Cmd("echo hello", Dir("/tmp"), Stdout, func(c *osexec.Cmd) {
		c.Stdout = &buf
	})
	assert.Equal(t, "/tmp", cmd.Dir)
	assert.Equal(t, &buf, cmd.Stdout)
}
