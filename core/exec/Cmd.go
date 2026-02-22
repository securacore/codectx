package exec

import (
	"os/exec"
	"strings"
)

type cmdOption func(*exec.Cmd)

func Cmd(cmd string, opts ...cmdOption) *exec.Cmd {
	var args []string
	var builder strings.Builder
	var inSingle, inDouble, inBacktick bool

	for _, ch := range cmd {
		switch {
		case ch == '\'' && !inDouble && !inBacktick:
			inSingle = !inSingle
			builder.WriteRune(ch)
		case ch == '"' && !inSingle && !inBacktick:
			inDouble = !inDouble
			builder.WriteRune(ch)
		case ch == '`' && !inSingle && !inDouble:
			inBacktick = !inBacktick
			builder.WriteRune(ch)
		case ch == ' ' && !inSingle && !inDouble && !inBacktick:
			if builder.Len() > 0 {
				args = append(args, builder.String())
				builder.Reset()
			}
		default:
			builder.WriteRune(ch)
		}
	}

	if builder.Len() > 0 {
		args = append(args, builder.String())
	}

	proc := exec.Command(args[0], args[1:]...)

	for _, fn := range opts {
		fn(proc)
	}

	return proc
}
