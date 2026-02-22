# Go

Go conventions for this repository. This document covers the language, CLI architecture, project structure, TUI patterns, and tooling. It governs all Go code in the project. All repository operations run through Just; this document covers Go code conventions, not workflow operations.

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Project Layout

The project follows a flat top-level structure with three primary directories.

<rules>

- `main.go` at the project root is the CLI entry point. It registers top-level commands and starts the application.
- `cmds/` contains CLI commands. Each command is a subdirectory with its own package.
- `core/` contains core system packages organized by domain. Each subdirectory is a domain package (e.g., `core/exec/`, `core/errs/`).
- Sub-packages within `core/` are permitted when a domain has distinct sub-domains (e.g., `core/sys/logging/`).
- Keep the directory structure flat unless nesting earns its place. Do not create deep hierarchies for organizational purposes alone.
- Do not use `internal/` or `pkg/` directories.

</rules>

## Package Conventions

<rules>

- Package name matches the directory name exactly. A package in `cmds/version/` is `package version`.
- Package names are lowercase, single-word, no underscores, no mixedCaps.
- Do not use prefix conventions. No `cmd_`, `pkg_`, or similar prefixes on package names.
- Use import aliases only at the call site when an actual name collision occurs.
- One package per directory.

</rules>

## Command Structure

Commands use `urfave/cli/v3`. Each command is a self-contained package under `cmds/`.

<rules>

- Each command lives in `cmds/<name>/` as its own package.
- The package exports a single `Command` variable of type `*cli.Command`.
- The entry file for a command package is `main.go`.
- Subcommands are nested by adding to the parent command's `Commands` slice.
- `main.go` at the project root registers all top-level commands.
- Propagate `context.Context` through the CLI framework. Do not create separate context chains.

</rules>

## File Naming

File naming follows a one-exported-symbol-per-file convention with PascalCase filenames.

<rules>

- Name each file after the primary exported symbol it contains. `Cmd.go` exports the `Cmd` function. `Error.go` exports the `Error` type.
- Use PascalCase for file names that export a symbol.
- One exported symbol per file. Private types, helpers, and methods that support the exported symbol live in the same file.
- Test files use the `_test.go` suffix with PascalCase matching the file under test: `Cmd_test.go` tests `Cmd.go`.
- `main.go` is the entry point file for a package. It contains the package's primary export or initialization logic.
- `doc.go` is permitted for package-level documentation when the package warrants it.
- Files that contain only unexported helpers shared across the package use lowercase naming (e.g., `helpers.go`).
- When multiple tightly coupled exported symbols cannot be meaningfully separated, they may share a file. Name the file after the primary symbol.
- Do not force the one-symbol-per-file pattern when it does not serve the design. Adapt the convention to fit the system, not the system to fit the convention.

</rules>

## Naming Conventions

<rules>

- Exported identifiers use PascalCase: `RunCommand`, `Version`, `NewClient`.
- Unexported identifiers use camelCase: `parseFlags`, `configPath`, `buildArgs`.
- Acronyms are fully capitalized: `HTTP`, `URL`, `ID`, `API`, `TUI`. Write `HTTPClient`, not `HttpClient`. Write `userID`, not `userId`.
- No getter prefix. Use `Name()` for getters, `SetName()` for setters. The asymmetry is intentional: methods that return values are getters by definition.
- Single-method interfaces use the `-er` suffix: `Reader`, `Writer`, `Formatter`, `Builder`.
- Exported constants use PascalCase: `MaxRetries`, `DefaultTimeout`.
- Unexported constants use camelCase: `maxRetries`, `defaultTimeout`.
- Boolean variables and functions use `is`, `has`, `can` prefixes when the prefix aids clarity: `isValid`, `hasPermission`, `canRetry`.

</rules>

## Type Conventions

<rules>

- Return concrete types from functions. Accept interfaces as parameters when flexibility is needed.
- Order struct fields with exported fields first, then unexported fields, grouped logically by purpose.
- Use `type` definitions to create domain-specific types (e.g., `type cmdOption func(*exec.Cmd)`).
- Do not use type aliases unless wrapping an external type for decoupling.
- The functional options pattern is the preferred configuration pattern. Define an unexported function type and provide exported option functions that return it.

</rules>

## Error Handling

Application-level errors live in `core/errs/`. This package builds on the standard library `errors` package and defines structured error types for the CLI.

<rules>

- Return errors from functions. Never panic in library or package code.
- `log.Fatal` is permitted only in `main()`.
- Wrap errors with context using `fmt.Errorf("context: %w", err)`.
- Define sentinel errors with `var ErrNotFound = errors.New("not found")` for known, matchable conditions.
- Use `errors.Is()` and `errors.As()` for programmatic error checking.
- The `core/errs/` package defines application error types with a `Kind` enum for categorization (not found, invalid, permission, internal).
- Application error types implement the `error` interface and support `Unwrap()` for chain inspection.
- The package name is `errs`, not `errors`, to avoid shadowing the standard library.
- Detailed CLI error strategy (exit codes, user-facing formatting) is deferred until the CLI matures. The `Kind` enum provides the foundation for mapping errors to exit codes and formatted output when that strategy is defined.

</rules>

## Functions

<rules>

- Return early with guard clauses. Reduce nesting by handling error cases and preconditions at the top of the function.
- Use named return values only when they serve as documentation for the caller. Do not use named returns solely for naked return statements.
- Aim for a single return path when practical. Do not contort code structure to achieve it.
- Use the functional options pattern for configuration. This is the established pattern in `core/exec/` and applies to any constructor or factory that accepts optional configuration.

</rules>

## Testing

Testing uses `testify` for assertions.

<rules>

- Use `testify/assert` for soft assertions that allow the test to continue on failure.
- Use `testify/require` for hard assertions that stop the test on failure. Use `require` for preconditions and setup validation.
- Table-driven tests are the default pattern for any function with multiple input/output cases.
- Write tests in the same package as the code under test (white-box testing).
- Name test functions as `Test<Function>_<scenario>`: `TestCmd_handlesQuotedArgs`, `TestNew_returnsDefaultConfig`.
- Name test files with the `_test.go` suffix matching the file under test: `Cmd_test.go` for `Cmd.go`.

</rules>

## Dependencies

<rules>

- Evaluate well-established, trusted third-party packages before building custom solutions. Use the community standard when one exists (e.g., `urfave/cli` for CLI, `testify` for testing, `bubbletea` for TUI).
- When no well-established package exists, evaluate secondary and tertiary options. Weigh external packages against the cost and maintenance burden of a custom solution.
- Build custom only when external options do not meet requirements or introduce disproportionate complexity.
- Run `go mod tidy` after any dependency change.
- Do not vendor dependencies.

</rules>

## Formatting and Tooling

<rules>

- `gofmt` is non-negotiable. All Go code is formatted with `gofmt`.
- Use `goimports` for import group ordering: standard library, then external packages, then internal packages.
- Use `golangci-lint` for comprehensive linting. Install it via devbox.
- Compiler and tooling configuration is source of truth. Per [philosophy.md](../../foundation/philosophy.md), when this document and configuration conflict, configuration wins.

</rules>

## TUI Architecture

TUI components use Bubble Tea (Elm Architecture: Model-Update-View), Lip Gloss for styling, and Bubbles for pre-built components. All TUI packages are from the Charm ecosystem.

### Component Organization

Shared TUI components live in `core/tui/`, organized by concern.

<rules>

- `core/tui/component/` contains reusable UI components. Each component is a sub-package.
- `core/tui/style/` contains Lip Gloss style definitions, color palettes, and theme configuration.
- `core/tui/msg/` contains shared message types for cross-component communication.
- `core/tui/key/` contains shared key binding definitions.
- Command-specific TUI models live inline with the command package in `cmds/<name>/`.

</rules>

### Component File Structure

Every TUI component is a sub-package with a consistent file split.

<rules>

- Each component is a sub-package under `core/tui/component/<name>/`.
- `Model.go` contains the model struct, the `New()` constructor, and the `Init()` method.
- `Update.go` contains the `Update()` method.
- `View.go` contains the `View()` method.
- Additional files are permitted for complex components (e.g., `Styles.go` for component-specific styles, helper files for complex rendering logic).
- Shared components return their concrete type from `Update()`. They are child components composed by parents.
- Command-level models return `tea.Model` from `Update()` to satisfy the `tea.Model` interface for `tea.NewProgram`.

</rules>

### Composition Pattern

<rules>

- Parent models hold child component models as struct fields.
- Parent `Update()` methods propagate messages to child components by calling each child's `Update()` and collecting commands.
- Parent `View()` methods compose output by calling each child's `View()` and concatenating results.
- Use `tea.Batch()` to combine commands from multiple child component updates.

</rules>

### Styling with Lip Gloss

<rules>

- Define all colors using the oklch color space for perceptual uniformity. This aligns with the color system conventions in the Tailwind topic.
- Centralize the color palette in `core/tui/style/`. Individual components reference shared style definitions rather than defining ad-hoc colors.
- Use `lipgloss.NewStyle()` to build styles. Chain methods for readability.
- Support terminal width by responding to `tea.WindowSizeMsg` in components that need responsive layout.
- Define a theme type in `core/tui/style/` that holds the application's style set. Pass the theme to components that need it.

</rules>

### Inline Prompts

For simple interactive inputs (confirmations, form fields, selections) that do not require a full-screen TUI, use Charm's `huh` library.

<rules>

- Use `huh` directly in command `Action` functions. Do not wrap it in a custom abstraction.
- Reserve the full Bubble Tea Model-Update-View pattern for complex, stateful, full-screen interfaces.

</rules>
