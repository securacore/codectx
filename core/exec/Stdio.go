package exec

import (
	"os"
	"os/exec"
)

func Stdio(c *exec.Cmd) {
	c.Stdin = os.Stdin
	c.Stdout = os.Stdout
	c.Stderr = os.Stderr
}
