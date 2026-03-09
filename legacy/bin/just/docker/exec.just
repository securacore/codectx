alias x := exec

# Runs a command against the container
[group('docker')]
exec *args='':
  #!/usr/bin/env bash
  docker compose exec archlinux /bin/bash -c "{{args}}"
