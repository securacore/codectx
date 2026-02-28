# Plans Specification

Spec for the plans foundation document.

## Purpose

Implementation plans need structured lifecycle management so AI tools can triage work efficiently and engineers can track progress consistently. Without explicit conventions, plan status is scattered across commit messages, issue trackers, and ad-hoc notes. The plans foundation document codifies the create-read-update-delete lifecycle and introduces a lightweight state file (`plan.yml`) that enables AI triage without loading full plan content.

## Decisions

- **Two-file structure (README.md + plan.yml).** Separating the full plan content from the lightweight state enables AI tools to assess plan status with minimal token cost. Reading a 5-line YAML file is far cheaper than loading a multi-hundred-line plan document. Alternative considered: single-file with YAML frontmatter (rejected; frontmatter requires parsing mixed content and is less reliably extracted by AI tools).

- **Four status values (not_started, in_progress, completed, blocked).** This set covers the meaningful states for implementation work without over-engineering. Each status maps to a clear action: `not_started` means "start here", `in_progress` means "continue", `completed` means "done, archive", `blocked` means "resolve dependency first". Alternative considered: finer-grained statuses like `paused`, `reviewing`, `deferred` (rejected; additional statuses add complexity without proportional triage value).

- **Summary field for AI triage.** The 1-3 sentence summary in `plan.yml` gives AI tools enough context to decide whether to load the full plan. This is the primary interface for routine status checks. Alternative considered: relying on the README's first paragraph (rejected; requires loading the full file and parsing markdown structure).

- **`load: documentation` for this document.** Plan lifecycle management is a documentation task. Plans are created, updated, and reviewed as part of documentation workflows. Loading in code-only sessions wastes tokens. Alternative considered: `load: always` (rejected; most coding sessions do not involve plan management, and plans can be loaded on-demand when referenced).

- **Plans as foundation documentation.** Plan management is a cross-cutting process concern, not a technology-specific topic. Every project using codectx may have implementation plans regardless of its technology stack. Alternative considered: topic documentation (rejected; plans are not technology-specific).

## Dependencies

- [documentation](../../documentation/README.md): documentation organization and directory structure conventions
- [specs](../../specs/README.md): spec format referenced for plan spec subdirectories
