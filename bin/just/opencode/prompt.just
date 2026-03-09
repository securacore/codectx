# Use this to easily invoke the README.md files from
# the docs/prompts the repo uses.
[group("opencode")]
prompt name prompt="Run the prompt against the repo.":
  #!/usr/bin/env bash
  set -euo pipefail

  name="{{name}}"

  if [[ -z "$name" ]]; then
    echo "A plan name is required. Plan names are the"
    echo "names of one of the directories in the docs/plans/"
    echo "name."
    exit 1
  fi

  if ! command -v opencode &> /dev/null; then
    echo "Error: This command requires 'opencode' to be installed."
    echo ""
    echo "Please install OpenCode to use this command."
    echo "Visit: https://opencode.ai for installation instructions."
    exit 1
  fi

  prompt_path="{{justfile_directory()}}/docs/prompts/$name/README.md"

  if [[ ! -d "$prompt_path" ]]; then
    echo "Unable to find plan entrypoint:"
    echo "$prompt_path"
    exit 1
  fi

  prompt="{{prompt}} "
  prompt+="Do NOT add AI signatures or co-author attributions."

  cd "{{invocation_directory()}}"

  if ! opencode run "$prompt" -f "$prompt_path"; then
    echo ""
    echo "Error: OpenCode failed to generate commit."
    echo "Unstaging changes..."
    git reset
    exit 1
  fi
