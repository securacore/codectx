# Build the CLI binary with version injected from the latest git tag.
[group('root')]
build:
  #!/usr/bin/env bash
  set -euo pipefail
  tag="$(git tag --sort=-v:refname --list 'v*' | head -n 1)"
  version="${tag#v}"
  if [ -z "$version" ]; then
    version="dev"
  fi
  go build -ldflags "-s -w -X github.com/securacore/codectx/cmds/version.Version=${version}" -o codectx .
  echo "Built codectx ${version}"
