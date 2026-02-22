package exec

import (
	"os"
	"os/exec"
)

func Stdin(c *exec.Cmd) {
	c.Stdin = os.Stdin
}
