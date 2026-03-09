# Specs Specification

Spec for the specification documentation foundation document.

## Purpose

Without a standardized spec format, reasoning about documentation decisions is captured inconsistently or not at all. The spec template ensures every documentation directory includes a structured record of _why_ the documentation exists and _why_ it is organized the way it is. This enables AI tools to trace decisions, identify gaps, and make informed presumptions when clarification is needed.

## Decisions

- **Specs as mandatory, not optional.** Every documentation directory requires a `spec/README.md`. Without enforcement, specs are never written and the reasoning behind documentation is lost. Alternative considered: optional specs (rejected; optional means "never done" in practice, and missing reasoning creates blind spots for AI tools).

- **Four-section template (Purpose, Decisions, Dependencies, Structure).** This covers the minimum information needed to understand and reproduce a documentation decision. Purpose provides context. Decisions record reasoning. Dependencies map the relationship graph. Structure provides a file map. Alternative considered: freeform reasoning (rejected; inconsistent structure makes specs unreliable for AI parsing).

- **Reasoning only in Decisions, never restating conventions.** Specs record _why_, not _what_. The conventions document is authoritative for what the rule is. Duplicating the rule in the spec creates two sources of truth that drift apart. Alternative considered: including the convention alongside the reasoning (rejected; duplication invites drift).

- **`load: documentation` for this document.** The spec template is needed when authoring documentation, not when writing code. Alternative considered: `load: always` (rejected; unnecessary token cost for code-only sessions).

## Dependencies

- [documentation](../../documentation/README.md): references spec requirement for all documentation directories
