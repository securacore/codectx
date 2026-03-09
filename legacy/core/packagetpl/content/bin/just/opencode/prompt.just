# Use this to easily invoke the README.md files from
# the prompts/ the repo uses.
[private]
[group('opencode')]
prompt directory context="Run the prompt against the entire repo.":
  #!/usr/bin/env bash
  set -euo pipefail

  if ! command -v opencode &> /dev/null; then
    echo "Error: This command requires 'opencode' to be installed."
    echo ""
    echo "Please install OpenCode to use this command."
    echo "Visit: https://opencode.ai for installation instructions."
    exit 1
  fi

  prompt="{{context}} "
  prompt+="Do NOT add AI signatures or co-author attributions."

  cd "{{invocation_directory()}}"

  if ! opencode run "$prompt" -f "docs/prompts/{{directory}}/README.md"; then
    echo ""
    echo "Error: OpenCode failed to generate commit."
    echo "Unstaging changes..."
    git reset
    exit 1
  fi
