<p align="center">
  <img src=".assets/logo.svg" alt="codectx" width="180" />
</p>

<h1 align="center">codectx</h1>

<p align="center">
  A package manager for AI code documentation. Compile, share, and install structured context packages that give AI assistants deep understanding of your codebase conventions, architecture, and workflows.
</p>

<br />

## Install

**Shell (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh
```

**Homebrew:**

```bash
brew install securacore/tap/codectx
```

**Go:**

```bash
go install github.com/securacore/codectx@latest
```

Binaries are published for Linux and macOS on both `amd64` and `arm64`.
The install script detects your system architecture, downloads the
correct binary, and verifies its SHA256 checksum before installing to
`~/.local/bin`. Set `INSTALL_DIR` to override the install location.

## Usage

```bash
codectx init my-project    # Scaffold a new documentation package
codectx add react@org      # Install a package from GitHub
codectx compile            # Compile all packages into .codectx/
codectx link               # Symlink compiled output to tool-specific files
codectx search react       # Search the registry for packages
codectx version            # Print the installed version
```

The `codectx-` prefix is always implied. When you type `react@org`,
the CLI resolves it to `codectx-react` owned by `org` on GitHub.

## Development

### Prerequisites

- [devbox](https://www.jetify.com/devbox)
- [just](https://github.com/casey/just)

### Setup

```bash
just install
```

This installs devbox packages (Go, Docker, golangci-lint, lefthook),
runs `go mod download`, and installs git hooks via lefthook.

### Running the CLI

All arguments are forwarded into the Docker container:

```bash
just codectx compile
just codectx search react
```

Or connect to the container directly:

```bash
just connect
```

### Testing

```bash
go test ./... -count=1
```

### Linting

```bash
golangci-lint run ./...
```

Both run automatically on pre-commit via lefthook.

### Commands

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
│   └── version/           # codectx version
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
│   └── update/            # Background version update checker
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
Actions workflow that runs tests and lint before building binaries for
all platforms via GoReleaser.

```bash
bin/release          # Bump patch version (v0.1.0 -> v0.1.1)
bin/release minor    # Bump minor version (v0.1.0 -> v0.2.0)
bin/release major    # Bump major version (v0.1.0 -> v1.0.0)
```

The script requires a clean working tree on the `main` branch,
confirms the version bump, then pushes the tag to trigger the pipeline.

### Release pipeline

1. Tag push triggers `.github/workflows/release.yml`
2. Tests and lint run first as a gate
3. GoReleaser builds `linux/amd64`, `linux/arm64`, `darwin/amd64`,
   `darwin/arm64` binaries with version injected via ldflags
4. GitHub Release is created with tarballs and SHA256 checksums
5. Homebrew formula is automatically updated in `securacore/homebrew-tap`

### Secrets required

| Secret | Purpose |
|---|---|
| `GITHUB_TOKEN` | Automatic, used by GoReleaser to create releases |
| `HOMEBREW_TAP_TOKEN` | PAT with repo scope, pushes formula to `securacore/homebrew-tap` |

## Update Notifications

The CLI checks for newer versions in the background (once per 24 hours)
and displays a message after command output when an update is available.
This check never blocks command execution.

Disable with:

```bash
export CODECTX_NO_UPDATE_CHECK=1
```
