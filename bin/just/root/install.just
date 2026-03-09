# Install system packages.
[group("root")]
install:
  #!/usr/bin/env bash
  echo "Installing, please wait..."
  devbox install
  lefthook install
  go mod download
  just docker up
  just docker exec go mod tidy
  echo "Done"
