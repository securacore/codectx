# Handle starting the docker services for development.
[group('docker')]
up:
  #!/usr/bin/env bash
  docker compose up -d
