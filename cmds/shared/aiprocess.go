package shared

import (
	"os"
	"os/exec"
	"os/signal"
	"syscall"
)

// RunAIProcess launches an AI binary as a child process with stdio wired
// through and signals forwarded. This is the shared pattern used by the
// ide and normalize commands.
func RunAIProcess(binary string, args []string) error {
	cmd := exec.Command(binary, args...)
	cmd.Stdin = os.Stdin
	cmd.Stdout = os.Stdout
	cmd.Stderr = os.Stderr

	// Forward signals to the child process.
	sigCh := make(chan os.Signal, 1)
	signal.Notify(sigCh, syscall.SIGINT, syscall.SIGTERM)
	go func() {
		for sig := range sigCh {
			if cmd.Process != nil {
				_ = cmd.Process.Signal(sig)
			}
		}
	}()

	err := cmd.Run()
	signal.Stop(sigCh)
	close(sigCh)

	return err
}
