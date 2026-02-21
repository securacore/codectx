# Clean the repo into it's initial state.
[group('root')]
[arg('destroy', long='destroy', value='true')]
clean destroy='false':
  #!/usr/bin/env bash
  targets=(\
    ".next" \
    ".devbox" \
    "node_modules" \
    ".DS_Store" \
  )

  echo "Cleaning cached packages, please wait..."
  devbox run -- nix store gc --extra-experimental-features nix-command

  echo "Cleaning, please wait..."
  for target in "${targets[@]}"; do
    [[ -e "$target" ]] && rm -rfv "$target"
  done

  if [[ "{{destroy}}" == "true" ]]; then
    echo "Destroying ALL devbox packages and reclaiming space, please wait..."
    nix-collect-garbage -d
  fi

  echo "Done"

