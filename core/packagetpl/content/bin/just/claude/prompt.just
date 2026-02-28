# Use this to easily invoke the README.md files from
# the prompts/ the repo uses.
[private]
[group('claude')]
prompt directory context="Run the prompt against the entire repo.":
  #!/usr/bin/env bash
  set -euo pipefail

  if ! command -v claude &> /dev/null; then
    echo "Error: This command requires 'claude' to be installed."
    echo ""
    echo "Please install Claude Code to use this command."
    echo "Visit: https://claude.ai/download for installation instructions."
    exit 1
  fi

  prompt="{{context}} "
  prompt+="@prompts/{{directory}}/README.md "
  prompt+="Do NOT add AI signatures or co-author attributions."

  if ! claude -p --verbose --output-format stream-json "$prompt" --allowedTools "Bash,Read,Edit,Write" \
    | jq --unbuffered -j '
        select(.type == "assistant")
        | [.message.content[] | select(.type == "text") | .text]
        | join("")
        | if . != "" then . + "\n" else empty end'; then
    echo ""
    echo "Error: Claude failed to generate commit."
    echo "Unstaging changes..."
    git reset
    exit 1
  fi
