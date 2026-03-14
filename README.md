<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".assets/logo.dark.svg">
    <source media="(prefers-color-scheme: light)" srcset=".assets/logo.light.svg">
    <img alt="codectx" src=".assets/logo.light.svg" width="180" />
  </picture>
</p>

<h1 align="center">codectx</h1>

<p align="center">
  A package manager, compiler, and package manager for AI-driven documentation and AI documentation distribution.
</p>

<p align="center">
  <picture>
    <source media="(prefers-color-scheme: dark)" srcset=".assets/knowledge-base-diagram.dark.svg">
    <source media="(prefers-color-scheme: light)" srcset=".assets/knowledge-base-diagram.light.svg">
    <img alt="How codectx compiles documentation packages into AI-optimized context" src=".assets/knowledge-base-diagram.light.svg" width="720">
  </picture>
</p>

---

For a full overview of what codectx does and how it works, see the
[product overview](docs/product/README.md).

## Install

**Shell (Linux / macOS):**

```bash
curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | sh
```

**Go:**

```bash
go install github.com/securacore/codectx@latest
```

Binaries are published for Linux and macOS on `amd64` and `arm64`.
The install script detects your architecture, downloads the correct
binary, and verifies its SHA256 checksum. Set `INSTALL_DIR` to
override the install location.

## Development

### Prerequisites

- [devbox](https://www.jetify.com/devbox)
- [just](https://github.com/casey/just)

### Getting Started

> See what commands are avialable, if you want.

```bash
just
```

> Install and setup the project for development.

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

## Releasing

Releases are created by pushing a semver tag, which triggers a GitHub
Actions workflow that runs tests and lint before building binaries for
all platforms via GoReleaser.

```bash
# Two ways to generate a patch version update.
just release          # Bump patch version (v0.1.0 -> v0.1.1)
just release patch    # Bump patch version (v0.1.0 -> v0.1.1)

# How to increment minor and major versions.
just release minor    # Bump minor version (v0.1.0 -> v0.2.0)
just release major    # Bump major version (v0.1.0 -> v1.0.0)
```

The script requires a clean working tree on the `main` branch,
confirms the version bump, then pushes the tag to trigger the pipeline.

### Release Pipeline

1. Tag push triggers `.github/workflows/release.yml`
2. Tests and lint run first as a gate
3. GoReleaser builds `linux/amd64`, `linux/arm64`, `darwin/amd64`,
   `darwin/arm64` binaries with version injected via ldflags
4. GitHub Release is created with tarballs and SHA256 checksums

## Feedback

For any feedback or issues, please open an issue against the repo or contact the author (Jon Tech). Feedback is welcome to improve the CLI over time as features are added and refined.
