# Go Specification

Spec for the Go conventions documentation. For the conventions themselves, see [README.md](../README.md).

## Purpose

Go is the implementation language for the codectx CLI. Without explicit standards, every session makes independent decisions about project structure, package naming, file organization, command patterns, error handling, and TUI architecture. This documentation codifies those decisions so AI agents and engineers produce consistent, maintainable Go code across the project.

## Decisions

- **`cmds/` for commands, not `cmd/` or `internal/cmd/`.** The plural form `cmds/` is descriptive and distinguishes the directory from Go's conventional `cmd/` which typically holds multiple binaries in a multi-binary repository. This project has a single binary; `cmds/` holds subcommand packages, not separate binaries. Alternative considered: `internal/cmd/` (rejected; `internal/` is not used in this project, and the commands are the primary organizational unit, not an implementation detail to hide).

- **`core/` for system packages, not `internal/` or `pkg/`.** `core/` communicates that these packages are the foundational system capabilities. `internal/` is Go's access control mechanism, but this project is a single binary with no external consumers, making import restriction irrelevant. `pkg/` implies packages intended for external use, which is the opposite of the intent. Alternative considered: `internal/` (rejected; solves a problem this project does not have). Alternative considered: `pkg/` (rejected; misleading for a single-binary project).

- **Package name matches directory name.** Standard Go convention from Effective Go. Prefixing package names (e.g., `cmd_version` for `cmds/version/`) is non-idiomatic, triggers linter warnings, and creates friction with every Go tool and library. Import aliases handle the rare collision case at the call site. Alternative considered: `cmd_` prefix convention (rejected; non-idiomatic, linter friction, unnecessary).

- **PascalCase file naming with one symbol per file.** Creates a 1:1 mapping between file names and exported symbols. An AI agent or engineer looking for the `Cmd` function knows to open `Cmd.go`. This departs from Go's typical lowercase file naming but provides superior navigability, especially for AI-driven development where symbol-to-file lookup is a frequent operation. Alternative considered: lowercase with underscores (rejected; loses the direct symbol-to-file mapping). Alternative considered: one file per package (rejected; does not scale).

- **Functional options pattern as default configuration pattern.** Avoids parameter explosion in constructors, is self-documenting, and composes cleanly. The pattern is already established in `core/exec/` and provides a consistent API style across the project. Alternative considered: config structs (rejected; less composable, requires knowing all options upfront). Alternative considered: builder pattern with method chaining (rejected; mutable state, less idiomatic Go).

- **`core/errs/` for application errors.** Centralizes error types so the entire CLI uses a consistent error vocabulary. The `Kind` enum enables future mapping to exit codes and user-facing formatting without requiring changes to every command. Named `errs` to avoid shadowing the standard library `errors` package. Alternative considered: errors defined per-package (rejected; inconsistent error handling across commands, no central place to map errors to exit codes). Alternative considered: `core/errors/` (rejected; shadows stdlib, requires import aliases everywhere).

- **`testify` over standard `testing` package.** `testify` is the most widely used Go testing library and the community standard. It reduces assertion boilerplate, provides clear failure messages, and separates soft assertions (`assert`) from hard assertions (`require`). Alternative considered: standard `testing` only (rejected; excessive boilerplate for assertion-heavy tests). Alternative considered: other test libraries (rejected; none match testify's adoption, documentation, and ecosystem support).

- **`golangci-lint` over `go vet`.** `golangci-lint` runs `go vet` plus dozens of additional linters in a single pass. It catches more issues (unused code, complexity, style violations, potential bugs) and is configurable. Alternative considered: `go vet` only (rejected; insufficient for comprehensive code quality enforcement).

- **Bubble Tea for TUI.** Bubble Tea is the undisputed standard for Go terminal user interfaces. It uses the Elm Architecture (Model-Update-View), which provides a clean separation of state, logic, and rendering. The Charm ecosystem (Bubble Tea, Lip Gloss, Bubbles, huh) provides a complete toolkit. Alternative considered: `tview` (rejected; older design, less composable, different architectural model). Alternative considered: custom TUI framework (rejected; massive effort, no advantage over Charm).

- **Split-file TUI components (Model.go, Update.go, View.go).** TUI component implementations can grow complex. A consistent file split across all components provides predictable structure regardless of component complexity. An AI agent always knows where to find the update logic or the view rendering for any component. Alternative considered: single file per component (rejected; does not scale for complex components, inconsistent structure when some components are split and others are not). The consistent split is preferred even for simple components to maintain uniformity.

- **`huh` for inline prompts.** Charm's `huh` library handles forms, confirmations, and simple interactive inputs without requiring the full Bubble Tea Model-Update-View setup. Using it directly in command actions keeps simple interactions simple. Alternative considered: wrapping huh in a custom abstraction (rejected; adds indirection without benefit). Alternative considered: using full Bubble Tea for all interactions (rejected; unnecessary complexity for simple prompts).

- **Three-tier dependency hierarchy.** Prevents both NIH syndrome (building everything from scratch) and dependency bloat (importing packages for trivial operations). Well-established packages like `urfave/cli`, `testify`, and `bubbletea` have proven reliability and maintenance. Building custom is reserved for cases where external options genuinely do not fit. Alternative considered: minimize all external dependencies (rejected; reinvents well-solved problems). Alternative considered: no dependency policy (rejected; leads to inconsistent decisions across sessions).

- **`Name()`/`SetName()` over `GetName()`/`SetName()`.** Standard Go convention from Effective Go. The getter is the common operation and gets the shorter name. The `Set` prefix marks the mutating action; the asymmetry is the signal. `golangci-lint` flags `Get` prefixes on getters as non-idiomatic. Alternative considered: `GetName()`/`SetName()` for explicit symmetry (rejected; fights the linter, the standard library, and every Go dependency).

## Dependencies

- `go.mod`: module and dependency configuration
- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): guiding principles (referenced for "Configuration is Truth" and "Abstractions Must Earn Their Place")
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows

## Structure

- `README.md`: Go conventions (the actionable rules)
- `spec/README.md`: this file; reasoning and decisions behind the conventions
