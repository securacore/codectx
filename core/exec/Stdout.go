package exec

import (
	"os"
	"os/exec"
)

func Stdout(c *exec.Cmd) {
	c.Stdout = os.Stdout
}
