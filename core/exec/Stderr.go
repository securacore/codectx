package exec

import (
	"os"
	"os/exec"
)

func Stderr(c *exec.Cmd) {
	c.Stderr = os.Stderr
}
