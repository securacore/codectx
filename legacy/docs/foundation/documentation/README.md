# Documentation

This document defines how documentation is managed, organized, and written in this repository. For markdown formatting conventions, see [markdown.md](../markdown/README.md).

## Audience and Purpose

Documentation in this repository is written for AI agents first and engineers second. AI assistants consume these files as instructions; engineers use them as reference material. Foundational docs are loaded into every AI session, so consistent structure, predictable formatting, and clear section boundaries are essential for reliable parsing.

Documentation describes _approach and process_, never implementation. It answers "how do we build things here?" not "how does this specific code work?" This distinction is the foundation of every rule in this file.

Each document covers one cohesive topic so AI can selectively load only what is relevant. This keeps token usage low and context focused. A guide on building React components does not embed utility conventions inline. It references the utility guide, which AI can load if and when needed.

For cross-model authoring conventions that ensure documentation works across AI model tiers, see [ai-authoring.md](../ai-authoring/README.md).

## Documentation Structure

Documentation is organized in two tiers: foundational and topic-specific.

### Foundational Documentation

Foundational documents live in `docs/foundation/`. They govern how we work across the entire repository: philosophy, formatting conventions, documentation strategy. Each foundation document has a `load` field in `docs/metadata.yml` that controls when it is loaded into AI context: `always` for every session, `documentation` for sessions that involve writing or editing documentation. All foundation documents change only when the approach itself changes.

Foundational files follow the naming conventions in [markdown.md](../markdown/README.md).

### Topic Documentation

Topic documentation lives in subdirectories of `docs/topics/`, organized by technology, tool, or domain (e.g., `docs/topics/typescript/`, `docs/topics/react/`, `docs/topics/tailwind/`). Directory names use lowercase kebab-case.

Every topic directory has a `README.md` as its entry point. For straightforward topics, `README.md` is the entire documentation. For complex topics, `README.md` serves as a concordance that outlines scope and links to child files covering individual subtopics or steps.

A topic directory is warranted when a technology, tool, or domain has conventions specific enough to stand alone. Examples: a language with its own style rules, a framework with component patterns, a deployment tool with workflow conventions. Not every convention warrants its own directory. Create one when a topic has enough distinct rules to stand alone.

Every topic directory starts with `README.md` as the sole document. As the topic grows and the content becomes large enough to warrant splitting, break all sections into individual kebab-case `.md` files within the directory simultaneously. Do not split incrementally (one file at a time); split all at once so the structure is consistent. After splitting, `README.md` becomes a concordance that outlines scope and links to child files. The threshold for splitting is a judgment call based on document length, section independence, and whether selective loading would benefit AI context management.

Within any tier, split documents based on cohesion: when a document begins to cover multiple distinct concerns, break it into separate files that can be independently loaded and referenced.

### Specification Documentation

Topic directories may contain a `spec/` subdirectory that captures the reasoning behind the documentation. The spec template is defined in [specs.md](../specs/README.md).

## Timeless Content

Documentation must remain valid independent of the current state of application code. This means:

<rules>

- **No implementation samples.** Never include code that mirrors or references application code. When the code changes, the documentation becomes a liability.
- **Conceptual examples only.** When a sample aids understanding, it must illustrate the _concept_, not the application. Samples are supplementary. They exist to convey an idea, not to demonstrate how specifics are done in this codebase.
- **Configuration references, not duplication.** When documentation needs to reference configuration, point to the configuration file itself. Do not copy configuration values into documentation as examples. Per [philosophy.md](../philosophy/README.md), when documentation and configuration conflict, configuration wins. Avoid creating opportunities for conflict.

</rules>

## Code Comments vs. Documentation

Code comments and documentation files serve different purposes:

<rules>

- **Code comments** explain _intent_: why this specific code exists and what it is meant to accomplish. They are local to the code they annotate.
- **Documentation files** explain _process_: the conventions, patterns, and decisions that govern how code is written. They are independent of any specific code location.

If information is only relevant when reading a particular piece of code, it belongs in a comment. If it governs how code is written across the project, it belongs in documentation.

</rules>
