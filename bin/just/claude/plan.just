# Use this to easily invoke the README.md files from
# the docs/plans the repo uses.
[group("claude")]
plan name prompt="Run a plan against the repo.":
  #!/usr/bin/env bash
  set -euo pipefail

  name="{{name}}"

  if [[ -z "$name" ]]; then
    echo "A plan name is required. Plan names are the"
    echo "names of one of the directories in the docs/plans/"
    echo "directory."
    exit 1
  fi

  if ! command -v claude &> /dev/null; then
    echo "Error: This command requires 'claude' to be installed."
    echo ""
    echo "Please install Claude Code to use this command."
    echo "Visit: https://claude.ai/download for installation instructions."
    exit 1
  fi

  plan_path="{{justfile_directory()}}/docs/plans/$name/README.md"

  if [[ ! -d "$plan_path" ]]; then
    echo "Unable to find plan entrypoint:"
    echo "$plan_path"
    exit 1
  fi

  plan="{{prompt}} "
  plan+="@$plan_path "
  plan+="Do NOT add AI signatures or co-author attributions."

  if ! claude -p --verbose --output-format stream-json "$plan" --allowedTools "Bash,Read,Edit,Write" \
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
