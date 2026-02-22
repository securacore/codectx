package exec

import "os/exec"

func Dir(dir string) cmdOption {
	return func(c *exec.Cmd) {
		c.Dir = dir
	}
}
