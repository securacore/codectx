package ai

// Claude Code provider.
//
// Phase 1: detection only. The claude binary is checked via exec.LookPath
// in the standard Detect flow.
//
// Phase 2 will add Generate() support using the non-interactive mode:
//   claude --print "<prompt>"
//
// The --print flag sends a single prompt and prints the response to
// stdout without entering the interactive REPL.
