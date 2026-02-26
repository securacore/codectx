package cmdx

import (
	"context"
	"os"
	"path/filepath"
	"testing"

	"github.com/stretchr/testify/assert"
	"github.com/stretchr/testify/require"
	"github.com/urfave/cli/v3"
)

func TestCommand_metadata(t *testing.T) {
	assert.Equal(t, "cmdx", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Len(t, Command.Commands, 5, "should have 5 subcommands: encode, decode, stats, validate, roundtrip")

	names := make(map[string]bool)
	for _, sub := range Command.Commands {
		names[sub.Name] = true
	}
	assert.True(t, names["encode"], "should have encode subcommand")
	assert.True(t, names["decode"], "should have decode subcommand")
	assert.True(t, names["stats"], "should have stats subcommand")
	assert.True(t, names["validate"], "should have validate subcommand")
	assert.True(t, names["roundtrip"], "should have roundtrip subcommand")
}

// testdataPath returns the path to a testdata file in core/cmdx/testdata.
func testdataPath(name string) string {
	// Navigate from cmds/cmdx/ to core/cmdx/testdata/.
	return filepath.Join("..", "..", "core", "cmdx", "testdata", name)
}

func TestEncode_subcommand_writesStdout(t *testing.T) {
	// Capture stdout by redirecting to a temp file.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "encode", testdataPath("simple.md")})

	_ = w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err)

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	_ = r.Close()

	output := string(buf[:n])
	assert.Contains(t, output, "@CMDX v1", "stdout should contain CMDX output")
}

func TestEncode_subcommand_writesFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.cmdx")

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "encode", "-o", outPath, testdataPath("simple.md")})
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.Contains(t, string(data), "@CMDX v1", "output file should contain CMDX output")
}

func TestDecode_subcommand_writesStdout(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "decode", testdataPath("api_docs.cmdx")})

	_ = w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err)

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	_ = r.Close()

	output := string(buf[:n])
	assert.Contains(t, output, "User Management API", "stdout should contain decoded markdown")
}

func TestStats_subcommand_output(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "stats", testdataPath("api_docs.md")})

	_ = w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err)

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	_ = r.Close()

	output := string(buf[:n])
	assert.Contains(t, output, "Original", "should contain original size")
	assert.Contains(t, output, "Compressed", "should contain compressed size")
	assert.Contains(t, output, "Byte savings", "should contain byte savings")
	assert.Contains(t, output, "Token savings", "should contain token savings")
}

func TestValidate_validFile(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "validate", testdataPath("api_docs.cmdx")})
	assert.NoError(t, err, "valid CMDX should exit without error")
}

func TestValidate_invalidFile(t *testing.T) {
	// Create a temp file with invalid CMDX.
	tmpDir := t.TempDir()
	invalidPath := filepath.Join(tmpDir, "invalid.cmdx")
	err := os.WriteFile(invalidPath, []byte("not valid cmdx"), 0644)
	require.NoError(t, err)

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err = app.Run(context.Background(), []string{"test", "cmdx", "validate", invalidPath})
	assert.Error(t, err, "invalid CMDX should exit with error")
}

func TestRoundtrip_subcommand_success(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "cmdx", "roundtrip", testdataPath("simple.md")})
	assert.NoError(t, err, "lossless file should exit without error")
}
