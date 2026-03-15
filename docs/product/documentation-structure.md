# Documentation Structure

codectx organizes documentation into four content types, each serving a distinct purpose. This structure is consistent across local projects and published packages.

---

## Content Types

### Foundation

Technology-agnostic guidance that survives a tech stack change. Engineering principles, architectural philosophy, coding standards, design conventions, general best practices.

**The portability test**: if this guidance would still apply after switching frameworks, it belongs in foundation.

Foundation documents are what the AI reads when it needs to decide between two valid implementation approaches. They establish which approach aligns with this project's values and engineering culture.

Examples:
- `foundation/coding-standards/` — naming conventions, error handling patterns, code organization
- `foundation/architecture-principles/` — separation of concerns, dependency rules, module boundaries
- `foundation/review-standards/` — PR review criteria, approval requirements, quality thresholds

### Topics

Technology-specific documentation. How specific systems, components, frameworks, or libraries work within this project. This is where the AI goes to understand what exists and how to work with it.

Examples:
- `topics/react/` — component patterns, hooks, state management, memoization
- `topics/nextjs/` — routing, server actions, middleware, configuration
- `topics/authentication/` — JWT management, OAuth flows, session handling
- `topics/postgres/` — schema conventions, migration patterns, query optimization

### Plans

Living documentation that tracks work in progress. Plans have the same structure as foundation and topics, but with a `plan.yml` state file that enables [resumable AI-driven development](plans.md).

As plans are executed, the AI updates the plan state. If it loses context, stops mid-task, or continues on another machine, it can resume where it left off. Plans reference which documentation they depend on so the AI can reload the right context efficiently.

Examples:
- `plans/auth-migration/` — multi-step migration from session-based to JWT auth
- `plans/api-v2/` — incremental API redesign across multiple services

### Prompts

Natural language scripts — pre-crafted instructions the AI should execute and follow. These codify tribal knowledge about how to instruct the AI effectively for specific task types.

Prompts with their `.spec.md` files become maintainable tools rather than magic incantations. When a prompt stops working well, the spec explains why it was written that way, enabling informed revision.

Examples:
- `prompts/code-review/` — instructions for AI-assisted code review
- `prompts/docs-audit/` — instructions for identifying documentation gaps
- `prompts/refactor/` — guidelines for safe refactoring with test validation

---

## Directory Conventions

### Naming

- Topic directories use **lowercase kebab-case**: `error-handling/`, `jwt-auth/`, `database-migrations/`
- The directory name is the human-readable description of its purpose
- Every topic directory must contain a **`README.md`** as its entry point (mirrors GitHub's directory rendering convention)
- Sub-topic breakdowns use lowercase kebab-case `.md` files: `refresh-tokens.md`, `middleware-chain.md`
- Reasoning files mirror their parent with `.spec.md` suffix: `README.spec.md`, `refresh-tokens.spec.md`

### Project Layout

```
docs/                              # Default root (configurable)
  foundation/
    coding-standards/
      README.md                    # Entry point
      README.spec.md               # Reasoning behind the standards
    architecture-principles/
      README.md
  topics/
    authentication/
      README.md                    # Entry point for auth topic
      jwt-tokens.md                # Sub-topic breakdown
      oauth.md                     # Sub-topic breakdown
      README.spec.md               # Reasoning behind auth decisions
      jwt-tokens.spec.md           # Reasoning behind JWT design
    react/
      README.md
      components.md
      hooks.md
      state.md
  plans/
    auth-migration/
      README.md                    # Plan documentation
      plan.yml                     # State tracking (resumable)
  prompts/
    code-review/
      README.md                    # AI review instructions
      README.spec.md               # Why these instructions work
  system/                          # Compiler behavior (editable)
    foundation/
      documentation-protocol/
        README.md                  # How AI tools use codectx
      history/
        README.md                  # How AI tools use codectx history
    topics/
      taxonomy-generation/
        README.md                  # Alias generation instructions
      bridge-summaries/
        README.md                  # Bridge summary instructions
      context-assembly/
        README.md                  # Context assembly instructions
  codectx.yml                      # Project manifest
  codectx.lock                     # Dependency lock file
  .codectx/                        # Tooling state
    ai.yml                         # AI model config (checked in)
    ai.local.yml                   # API keys (gitignored)
    preferences.yml                # Compiler config (checked in)
    compiled/                      # All compiled output (gitignored)
    packages/                      # Installed packages (gitignored)
    history/                       # Query/generate history (gitignored)
```

The `docs/` root is configurable via the `root` field in `codectx.yml`. If `docs/` already contains non-codectx content, `codectx init` detects the conflict and prompts for an alternative.

---

## The .spec.md Convention

Every documentation file can have a corresponding `.spec.md` file that captures the reasoning behind the documentation.

### Why Specs Matter

When the AI encounters a situation the documentation doesn't explicitly cover, the `.spec.md` files give it the *intent* behind the instructions. This enables reasoning by analogy rather than guessing.

**Example**: `authentication/README.md` says "always use RS256 for JWT signing." The corresponding `README.spec.md` explains *why* — the team uses asymmetric signing so that services can verify tokens without access to the signing key, enabling a zero-trust inter-service architecture. When the AI encounters a new service that needs to validate tokens, the spec tells it what the signing strategy is *trying to achieve*, guiding correct implementation even for scenarios the docs don't explicitly cover.

### Specs for AI-Authored Documentation

When the AI generates or maintains documentation, `.spec.md` files are particularly valuable. They record the reasoning that produced each decision, creating an audit trail that both humans and AI can reference during maintenance and updates.

### How Specs Are Compiled

Spec files are compiled into their own chunk type (`spec:` prefix) and indexed in a separate BM25 index. This means:

- Searching for "how to implement authentication" returns instruction chunks
- Searching for "why we use RS256" returns reasoning chunks
- The two types never dilute each other's relevance scores
- The manifest cross-references spec chunks to their parent instruction chunks

---

## The System Directory

The `system/` directory contains documentation about the compiler itself, using the same four-directory structure (foundation, topics, plans, prompts). These files govern how the compiler's AI-driven passes behave.

### What Lives in System

- **`system/foundation/documentation-protocol/`** — Instructions for how AI tools interact with codectx
- **`system/foundation/history/`** — Instructions for how AI tools use codectx history commands
- **`system/topics/context-assembly/`** — Instructions for assembling session context
- **`system/topics/taxonomy-generation/`** — Instructions for generating taxonomy aliases (created as needed)
- **`system/topics/bridge-summaries/`** — Instructions for generating chunk boundary bridges (created as needed)
- **`system/plans/`** — Empty on init, available for compiler migration plans
- **`system/prompts/`** — Automation scripts for compiler-adjacent tasks

### How System Files Work

System files ship as sensible defaults on `codectx init` and belong to you from that moment on. Modifying these files changes the compiler's behavior on the next compilation.

System `.md` files (excluding `.spec.md`) are compiled into `compiled/system/` with the `sys:` prefix and indexed in their own BM25 index. This makes compiler documentation searchable — the AI can ask "how does the compiler generate aliases?" and get results. System `.spec.md` files are compiled into `compiled/specs/` alongside all other reasoning chunks.

### Never Published

The `system/` directory is never included in published packages. When a consumer installs a package, their local compiler processes the package content using their own `system/` instructions. One set of compilation rules per project, not per package.

---

## Project vs. Package

A **project** is the full documentation environment: authored docs, compiler configuration, AI instructions, installed packages, and compiled output.

A **package** is a publishable subset — just content. Packages contain only the four content directories plus a `codectx.yml` for identity:

```
my-package/
  codectx.yml          # Name, org, version, description, dependencies
  foundation/          # Optional
  topics/              # Optional
  plans/               # Optional
  prompts/             # Optional
```

No `system/` directory. No `.codectx/` directory. No compiler configuration. Package authoring has near-zero friction — write markdown, organize into the standard directories, publish. See [Package Manager](package-manager.md) for details.

---

[Back to overview](README.md)
