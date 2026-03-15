# CLI Reference

Complete command reference for codectx. All commands locate the project by walking up from the current directory looking for a `codectx.yml` file, similar to how git finds `.git/`.

---

## Core Workflow

### `codectx init [directory]`

Initialize a new documentation project. Defaults to `docs/` as the root directory. If `docs/` already exists with non-codectx content, prompts for an alternative root.

Creates the full directory structure, default `codectx.yml`, `ai.yml`, `preferences.yml`, default `system/` documentation, and `.gitignore` additions.

```bash
codectx init
codectx init ai-docs    # Use a custom root directory
```

### `codectx compile`

Run the full compilation pipeline. Processes all markdown files through parsing, stripping, chunking, indexing, taxonomy extraction, manifest generation, context assembly, and entry point linking.

```bash
codectx compile
codectx compile --incremental=false    # Force full recompilation
```

Output:

```
Compiled: 342 files -> 4,850 chunks (2,134,500 tokens)
Taxonomy: 12,847 terms, 48,203 aliases
Session: 28,450 / 30,000 tokens (94.8%)
Changes: 3 new, 7 modified, 332 unchanged
Time: 47.3s
```

Incremental compilation is enabled by default — only changed files are reprocessed.

### `codectx query "<search terms>"`

Search the compiled documentation. Returns ranked results from all three indexes (instructions, reasoning, system) fused via Reciprocal Rank Fusion.

```bash
codectx query "authentication token validation"
codectx query --top 20 "error handling patterns"
```

| Flag | Description |
|------|-------------|
| `--top N` | Number of results to return (default from `ai.yml results_count`) |

### `codectx generate "<chunk-ids>"`

Assemble specific chunks into a single reading document. Accepts `obj:`, `spec:`, and `sys:` prefixed chunk IDs. Groups content by type: instructions first, system second, reasoning last.

```bash
codectx generate "obj:a1b2c3.01,obj:a1b2c3.02,spec:f7g8h9.01"
codectx generate --file output.md "obj:a1b2c3.01,obj:a1b2c3.02"
codectx generate --no-cache "obj:a1b2c3.01"
```

| Flag | Description |
|------|-------------|
| `--file <path>` | Write document to a file instead of stdout |
| `--no-cache` | Bypass cache lookup and run the full assembly pipeline |

### `codectx prompt "<search terms>"`

Query and generate in a single atomic operation. Searches for relevant chunks and immediately assembles the top results into a reading document.

```bash
codectx prompt "jwt token refresh flow"
codectx prompt --top 5 --budget 2000 "error handling"
codectx prompt --file output.md "authentication patterns"
codectx prompt --delta "obj:a1b2c3.01" "middleware chain"
```

| Flag | Description |
|------|-------------|
| `--top N` | Number of query results to consider |
| `--budget N` | Maximum token count for the assembled document |
| `--file <path>` | Write document to a file instead of stdout |
| `--no-cache` | Bypass generate cache |
| `--delta <ids>` | Exclude specific chunk IDs from results (already loaded) |

---

## Package Management

### `codectx add <dependency>`

Add a package dependency to the project.

```bash
codectx add react-patterns@community
codectx add react-patterns@community --inactive
codectx add react-patterns@community --project    # Add only to project deps
codectx add react-patterns@community --package    # Add only to package deps
```

| Flag | Description |
|------|-------------|
| `--project` | Add to project dependencies only |
| `--package` | Add to package dependencies only |
| `--both` | Add to both project and package dependencies |
| `--inactive` | Add as inactive (installed but excluded from compilation) |
| `--show-uninstallable` | Show packages that can't be installed on this platform |

### `codectx remove <dependency>`

Remove a package dependency.

```bash
codectx remove react-patterns@community
codectx remove react-patterns@community --project
```

| Flag | Description |
|------|-------------|
| `--project` | Remove from project dependencies only |
| `--package` | Remove from package dependencies only |
| `--both` | Remove from both |

### `codectx install`

Install all dependencies declared in `codectx.yml`. Uses `codectx.lock` for deterministic resolution when available.

```bash
codectx install
```

### `codectx update`

Re-resolve all dependencies to latest compatible versions, update `codectx.lock`, download changed packages, and optionally recompile.

```bash
codectx update
codectx update --no-compile    # Update deps without recompiling
```

| Flag | Description |
|------|-------------|
| `--compile` | Force recompilation after update |
| `--no-compile` | Skip recompilation |

### `codectx search "<query>"`

Search for packages on GitHub using the `codectx-*` naming convention.

```bash
codectx search "react patterns"
codectx search --limit 20 "testing"
```

| Flag | Description |
|------|-------------|
| `--limit N` | Maximum number of results |
| `--show-uninstallable` | Include packages not available on this platform |

### `codectx publish`

Publish the current package to GitHub. Validates structure, tags the commit, and pushes.

```bash
codectx publish
codectx publish --validate    # Validate only, don't publish
```

| Flag | Description |
|------|-------------|
| `--validate` / `--dry-run` | Validate package structure without publishing |

### `codectx new package`

Scaffold a new package project with the standard directory structure and CI workflow.

```bash
codectx new package
```

---

## Session Context

### `codectx session add <reference>`

Add a document or package to the always-loaded session context. Reports token cost.

```bash
codectx session add foundation/coding-standards
codectx session add react-patterns@community/foundation/component-principles
codectx session add company-standards@acme
```

### `codectx session remove <reference>`

Remove an entry from the session context.

```bash
codectx session remove company-standards@acme
```

### `codectx session list`

List all always-loaded entries with token counts and budget utilization.

```bash
codectx session list
```

---

## Plans

### `codectx plan status [plan-name]`

Report the current state of a plan. Shows completion, current step, dependency hash status, and stored queries.

```bash
codectx plan status auth-migration
```

### `codectx plan resume [plan-name]`

Resume a plan by reconstructing context. Replays stored chunks if dependencies are unchanged, or reports drift and provides stored queries for re-execution.

```bash
codectx plan resume auth-migration
```

---

## History and Usage

### `codectx history queries`

Show recent query invocations with expanded terms, result counts, and staleness indicators.

### `codectx history chunks`

Show recent generate invocations with chunk IDs, token counts, and cache hit status.

### `codectx history show <hash>`

Print a previously generated document. Accepts a prefix match against the content hash.

### `codectx history clear`

Delete all history data. Requires interactive terminal confirmation.

### `codectx usage`

Display local and global usage metrics.

```bash
codectx usage
codectx usage --local     # Local machine only
codectx usage --global    # Project lifetime only
codectx usage --reset-local
```

| Flag | Description |
|------|-------------|
| `--local` | Show only local machine metrics |
| `--global` | Show only project lifetime metrics |
| `--reset-local` | Reset local `usage.yml` to zero without syncing |

---

## Project Management

### `codectx link`

Generate AI tool entry point files (CLAUDE.md, AGENTS.md, .cursorrules, copilot-instructions.md).

```bash
codectx link
codectx link --yes     # Skip confirmation prompts
codectx link --all     # Generate all entry point types
```

| Flag | Description |
|------|-------------|
| `--yes` | Skip interactive confirmation |
| `--all` | Generate all supported entry point files |

### `codectx repair`

Repair the project scaffold. Ensures all standard directories exist, restores missing system defaults, manages `.gitkeep` files, and re-merges `.gitignore` entries.

```bash
codectx repair
```

### `codectx version`

Print the current CLI version.

```bash
codectx version
```

### `codectx version bump [major|minor|patch]`

Bump the project version in `codectx.yml`.

```bash
codectx version bump patch    # 1.2.0 -> 1.2.1
codectx version bump minor    # 1.2.0 -> 1.3.0
codectx version bump major    # 1.2.0 -> 2.0.0
```

### `codectx self update`

Update the codectx binary to the latest release.

```bash
codectx self update
```

---

[Back to overview](README.md)
