# Contributing

## Prerequisites

- [devbox](https://www.jetify.com/devbox)
- [just](https://github.com/casey/just)

## Setup

```bash
just install
```

This installs devbox packages (Go, Docker, golangci-lint, lefthook),
runs `go mod download`, and installs git hooks via lefthook.

## Running the CLI

All arguments are forwarded into the Docker container:

```bash
just codectx compile
just codectx search react
```

Or connect to the container directly:

```bash
just connect
```

## Testing

```bash
go test ./... -count=1
```

## Linting

```bash
golangci-lint run ./...
```

Both run automatically on pre-commit via lefthook.

## Commands

```bash
just          # List all available commands
just docker   # List Docker-specific commands
```

## Project Structure

```
codectx
├── cmds/                  # CLI subcommands (one package per command)
│   ├── add/               # codectx add
│   ├── compile/           # codectx compile
│   ├── init/              # codectx init
│   ├── link/              # codectx link
│   ├── search/            # codectx search (includes interactive TUI)
│   ├── version/           # codectx version
│   └── watch/             # codectx watch
├── core/                  # Domain logic (no CLI dependencies)
│   ├── compile/           # Compilation engine, content-addressed objects
│   ├── config/            # codectx.yml config loading/writing
│   ├── errs/              # Typed error handling
│   ├── exec/              # Shell command execution
│   ├── link/              # Symlink management for tool integration
│   ├── lock/              # Lock file generation
│   ├── manifest/          # package.yml manifest parsing
│   ├── resolve/           # Package resolution, fetching, search
│   ├── schema/            # JSON schema validation and embedding
│   ├── update/            # Background version update checker
│   └── watch/             # Filesystem watching for live recompilation
├── ui/                    # Shared TUI components, styles, output helpers
├── bin/
│   ├── install            # Curl-able installer script
│   ├── release            # Semver tagging and release trigger
│   └── just/              # Modular just recipes
├── .github/workflows/     # CI/CD (release pipeline)
├── .goreleaser.yml        # Cross-platform build configuration
├── lefthook.yml           # Git hook definitions (pre-commit, pre-push)
├── .golangci.yml          # Linter configuration
└── main.go                # CLI entrypoint
```

## Releasing

Releases are created by pushing a semver tag, which triggers a GitHub
Actions workflow that runs tests and lint before building binaries via
GoReleaser.

```bash
bin/release          # Bump patch version (v0.1.0 -> v0.1.1)
bin/release minor    # Bump minor version (v0.1.0 -> v0.2.0)
bin/release major    # Bump major version (v0.1.0 -> v1.0.0)
```

The script requires a clean working tree on the `main` branch,
confirms the version bump, then pushes the tag to trigger the pipeline.

### Release Pipeline

1. Tag push triggers `.github/workflows/release.yml`
2. Tests and lint run first as a gate
3. GoReleaser builds `linux/amd64`, `linux/arm64`, `darwin/amd64`,
   `darwin/arm64` binaries with version injected via ldflags
4. GitHub Release is created with tarballs and SHA256 checksums

### Secrets Required

| Secret | Purpose |
|---|---|
| `GITHUB_TOKEN` | Automatic, used by GoReleaser to create releases |

## Update Notifications

The CLI checks for newer versions in the background (once per 24 hours)
and displays a message after command output when an update is available.
This check never blocks command execution.

Disable with:

```bash
export CODECTX_NO_UPDATE_CHECK=1
```
