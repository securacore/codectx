# Connect to the container
[group('root')]
connect:
  docker compose exec --workdir /home/user/project archlinux \
  /bin/bash -c \
  "(cd /workspace && go install .) && exec bash"
