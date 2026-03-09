# Runs args against Docker container CLI.
[group('docker')]
cli *args='':
  #!/usr/bin/env bash

  docker compose exec archlinux \
    /bin/bash -c \
    "cd /workspace && go install . && cd /home/user/project && codectx {{args}}"

  exit 0
