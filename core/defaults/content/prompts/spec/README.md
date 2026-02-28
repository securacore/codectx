# Prompts Specification

Spec for the prompts foundation document.

## Purpose

Without explicit lifecycle documentation, prompts are created ad-hoc with inconsistent structure, invoked through undocumented mechanisms, and left stale when no longer needed. The prompts foundation document codifies the full create-read-update-delete lifecycle so that AI tools and engineers manage prompts consistently.

## Decisions

- **Prompts as foundation documentation, not topic documentation.** Prompts are a cross-cutting concern that applies to every project using codectx, regardless of technology stack. They belong in `foundation/` alongside other process documentation (documentation, specs, ai-authoring) rather than as a topic. Alternative considered: topic documentation (rejected; topics are technology-specific, and prompt management is universal).

- **`load: documentation` for this document.** The prompt lifecycle guide is needed when creating, editing, or managing prompts. These are documentation tasks, not code implementation tasks. Loading it in code-only sessions wastes tokens. Alternative considered: `load: always` (rejected; most sessions do not involve prompt creation or management).

- **CRUD lifecycle structure.** Organizing by lifecycle operation (create, invoke, update, delete) matches how engineers and AI tools interact with prompts in practice. Each operation has a clear entry point and completion criterion. Alternative considered: organizing by concept (structure, format, invocation) (rejected; lifecycle ordering is more actionable).

- **Manifest registration documented inline.** The manifest format for prompt entries is documented within this foundation doc rather than in a separate document. Prompts have the simplest manifest entry type (id, path, description) and documenting it inline avoids a cross-reference hop for a straightforward concept.

## Dependencies

- [ai-authoring](../../ai-authoring/README.md): authoring conventions for prompt-specific patterns
- [documentation](../../documentation/README.md): documentation organization referenced for directory structure
