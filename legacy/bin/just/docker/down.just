# Kills and removes the docker container.
[group('docker')]
down:
  #!/usr/bin/env bash
  set -euo pipefail
  cfg="{{invocation_directory()}}/compose.yml"

  service="$(cat $cfg | yq '.services | keys[]' | tr -d '"')"
  status="$(docker compose ps --format=json $service | jq ".State" | tr -d '"')"

  if [[ "$status" == "running" ]]; then
    # Kill the service
    docker compose kill $service --remove-orphans
  fi

  # Remove container
  docker compose rm $service -f

  # Removes all images
  docker compose down --rmi all
