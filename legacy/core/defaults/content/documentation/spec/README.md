# Documentation Specification

Spec for the documentation foundation document.

## Purpose

Without explicit documentation conventions, every AI session and every engineer makes independent decisions about where to put documentation, how to organize it, and what level of detail to include. This creates drift, inconsistency, and fragmented knowledge. The documentation conventions codify the organizational rules so every contributor (human or AI) produces documentation that fits the existing structure.

## Decisions

- **AI-first audience.** AI assistants are the primary consumers in active sessions and need predictable structure for reliable parsing. Engineers benefit from the same clarity. Alternative considered: engineer-first (rejected; AI context loading is the higher-frequency use case and imposes stricter structural requirements).

- **Two-tier organization (foundation + topics).** Foundation docs govern cross-cutting concerns; topic docs cover specific technologies or domains. This separation prevents topic-specific detail from polluting foundational rules and allows selective loading. Alternative considered: flat structure (rejected; does not scale, mixes concerns). Alternative considered: deeper hierarchy (rejected; adds navigational complexity without proportional benefit).

- **Mandatory spec subdirectories.** Every documentation directory requires a `spec/README.md` that tracks the reasoning behind the documentation. This creates an audit trail that AI tools use to understand _why_ documentation exists, identify gaps, and make better decisions when clarification is needed. Alternative considered: optional specs (rejected; without enforcement, specs are never written, and the reasoning is lost).

- **Timeless content rule.** Documentation must not reference application code directly. This prevents documentation from becoming stale when code changes. Conceptual examples illustrate ideas without coupling to implementation. Alternative considered: allowing code references with update obligations (rejected; update obligations are forgotten and stale references mislead AI).

- **`load: documentation` for this document.** These conventions govern documentation authoring, not code implementation. Loading them in code-only sessions wastes tokens without benefit. Alternative considered: `load: always` (rejected; disproportionate token cost for sessions that never touch docs).

## Dependencies

- [philosophy](../../philosophy/README.md): guiding principles referenced for configuration-is-truth
- [markdown](../../markdown/README.md): formatting conventions referenced for file naming
- [ai-authoring](../../ai-authoring/README.md): cross-model authoring conventions
- [specs](../../specs/README.md): spec template referenced for specification documentation
