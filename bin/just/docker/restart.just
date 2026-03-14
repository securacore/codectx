# Runs a command against the container
[group('docker')]
restart:
  #!/usr/bin/env bash
  just docker down
  just docker up
