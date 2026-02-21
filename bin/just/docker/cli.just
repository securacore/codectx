# Runs args against Docker container CLI.
[group('docker')]
cli *args='':
  #!/usr/bin/env bash

  docker compose exec archlinux \
    /bin/bash -c \
    "go run main.go {{args}}"

  exit 0
