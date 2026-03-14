# Install system packages.
[group("root")]
install:
  #!/usr/bin/env bash
  run="devbox run --"

  echo "Installing, please wait..."
  devbox install

  echo "Installing git hooks, please wait..."
  $run lefthook install

  echo "Installing project dependencies, please wait..."
  $run go mod download

  echo "Starting/Re-starting development container, please wait..."
  just docker restart

  echo "Installing container dependencies, please wait..."
  just docker exec go mod tidy

  echo "Done"
