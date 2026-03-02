# Run all tests.
[group('root')]
test *args:
  go test ./... {{ args }}

# Run tests with coverage report.
[group('root')]
test-coverage:
  #!/usr/bin/env bash
  set -euo pipefail
  go test -coverprofile=coverage.out ./...
  go tool cover -func=coverage.out | tail -1
  echo ""
  echo "Full report: go tool cover -html=coverage.out"

# Run tests for a specific package.
[group('root')]
test-pkg pkg *args:
  go test ./{{ pkg }}/... {{ args }}

# Run the linter.
[group('root')]
lint:
  golangci-lint run ./...
