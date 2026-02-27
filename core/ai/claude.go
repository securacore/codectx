package ai

// Claude Code provider.
//
// Detection only. The claude binary is checked via exec.LookPath
// in the standard Detect flow.
//
// For full streaming AI integration (documentation authoring), see core/llm/
// which wraps the Claude CLI using --output-format stream-json.
