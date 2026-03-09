# Search and optionally add a devbox CLI package.
[group("root")]
search *argv="":
  #!/usr/bin/env bash
  set -euo pipefail

  if [[ -z "{{argv}}" ]]; then
    echo "Usage: just search <term>"
    exit 1
  fi

  results="$(devbox search --show-all {{argv}} 2>&1)"

  packages="$(echo "$results" | grep '^\* ' | sed 's/^\* \([^ ]*\).*/\1/')"

  if [[ -z "$packages" ]]; then
    echo "No packages found."
    exit 0
  fi

  selected="$(printf '%s\n' $packages | fzf --prompt="Select package to add: " --header="Press ESC to cancel")" || {
    echo "No package selected."
    exit 0
  }

  if [[ -z "$selected" ]]; then
    echo "No package selected."
    exit 0
  fi

  echo "Adding $selected..."
  devbox add "$selected"
