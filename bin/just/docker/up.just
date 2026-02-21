# Handle starting the docker services for development.
[group('docker')]
up:
  docker compose up -d
