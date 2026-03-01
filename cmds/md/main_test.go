package md

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
	assert.Equal(t, "md", Command.Name)
	assert.NotEmpty(t, Command.Usage)
	assert.Len(t, Command.Commands, 3, "should have 3 subcommands: encode, stats, roundtrip")

	names := make(map[string]bool)
	for _, sub := range Command.Commands {
		names[sub.Name] = true
	}
	assert.True(t, names["encode"], "should have encode subcommand")
	assert.True(t, names["stats"], "should have stats subcommand")
	assert.True(t, names["roundtrip"], "should have roundtrip subcommand")
}

// testdataPath returns the path to a testdata file in core/md/testdata.
func testdataPath(name string) string {
	// Navigate from cmds/md/ to core/md/testdata/.
	return filepath.Join("..", "..", "core", "md", "testdata", name)
}

func TestEncode_subcommand_writesStdout(t *testing.T) {
	// Capture stdout by redirecting to a temp file.
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "md", "encode", testdataPath("simple.md")})

	_ = w.Close()
	os.Stdout = oldStdout

	require.NoError(t, err)

	buf := make([]byte, 65536)
	n, _ := r.Read(buf)
	_ = r.Close()

	output := string(buf[:n])
	// Output should be valid compact markdown.
	assert.NotEmpty(t, output, "stdout should contain encoded output")
}

func TestEncode_subcommand_writesFile(t *testing.T) {
	tmpDir := t.TempDir()
	outPath := filepath.Join(tmpDir, "output.md")

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "md", "encode", "-o", outPath, testdataPath("simple.md")})
	require.NoError(t, err)

	data, err := os.ReadFile(outPath)
	require.NoError(t, err)
	assert.NotEmpty(t, string(data), "output file should contain encoded output")
}

func TestStats_subcommand_output(t *testing.T) {
	oldStdout := os.Stdout
	r, w, _ := os.Pipe()
	os.Stdout = w

	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "md", "stats", testdataPath("api_docs.md")})

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

func TestRoundtrip_subcommand_success(t *testing.T) {
	app := &cli.Command{
		Commands: []*cli.Command{Command},
	}
	err := app.Run(context.Background(), []string{"test", "md", "roundtrip", testdataPath("simple.md")})
	assert.NoError(t, err, "lossless file should exit without error")
}
