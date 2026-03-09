# Documentation

How documentation is managed, organized, and written in this project. For markdown formatting conventions, see [markdown](../markdown/README.md).

## Audience and Purpose

Documentation is written for AI agents first and engineers second. AI assistants consume these files as instructions; engineers use them as reference material. Foundation docs are loaded into AI sessions, so consistent structure, predictable formatting, and clear section boundaries are essential for reliable parsing.

Documentation describes _approach and process_, never implementation. It answers "how do we build things here?" not "how does this specific code work?" This distinction is the foundation of every rule in this file.

Each document covers one cohesive topic so AI can selectively load only what is relevant. This keeps token usage low and context focused. A guide on one technology does not embed conventions for another technology inline. It references the other guide, which AI can load if and when needed.

For cross-model authoring conventions that ensure documentation works across AI model tiers, see [ai-authoring](../ai-authoring/README.md).

## Documentation Structure

Documentation is organized in two tiers: foundational and topic-specific.

### Foundational Documentation

Foundational documents live in `docs/foundation/`. They govern how work is done across the entire project: philosophy, formatting conventions, documentation strategy. Foundation documents change only when the approach itself changes.

Foundation entries in the manifest have a `load` field controlling when they are loaded into AI context: `always` for every session, `documentation` for sessions that involve writing or editing documentation.

### Topic Documentation

Topic documentation lives in subdirectories of `docs/topics/`, organized by technology, tool, or domain. Directory names use lowercase kebab-case.

Every topic directory has a `README.md` as its entry point. For straightforward topics, `README.md` is the entire documentation. For complex topics, `README.md` serves as a concordance that outlines scope and links to child files covering individual subtopics.

A topic directory is warranted when a technology, tool, or domain has conventions specific enough to stand alone. Not every convention warrants its own directory. Create one when a topic has enough distinct rules to stand alone as a cohesive unit.

Every topic directory starts with `README.md` as the sole document. As the topic grows and the content becomes large enough to warrant splitting, break all sections into individual kebab-case `.md` files within the directory simultaneously. Do not split incrementally; split all at once so the structure is consistent. After splitting, `README.md` becomes a concordance that outlines scope and links to child files.

### Specification Documentation

Every documentation directory must contain a `spec/` subdirectory with a `README.md` that captures the reasoning behind the documentation. The spec template is defined in [specs](../specs/README.md). Specs track the thinking, reasoning, and decisions that went into how the documentation was created. This enables AI tools and engineers to understand gaps, trace decisions, and make better presumptions when clarification is needed.

<rules>

- Every foundation directory requires a `spec/README.md`.
- Every topic directory requires a `spec/README.md`.
- Specs record _why_ the documentation exists and _why_ it is structured the way it is.
- Specs do not restate the conventions. The conventions document is the authoritative source for _what_ the rule is. The spec records _why_ that rule exists.

</rules>

## Timeless Content

Documentation must remain valid independent of the current state of application code.

<rules>

- **No implementation samples.** Never include code that mirrors or references application code. When the code changes, the documentation becomes a liability.
- **Conceptual examples only.** When a sample aids understanding, it must illustrate the _concept_, not the application. Samples are supplementary. They exist to convey an idea, not to demonstrate how specifics are done in this codebase.
- **Configuration references, not duplication.** When documentation needs to reference configuration, point to the configuration file itself. Do not copy configuration values into documentation. Per [philosophy](../philosophy/README.md), when documentation and configuration conflict, configuration wins.

</rules>

## Code Comments vs. Documentation

<rules>

- **Code comments** explain _intent_: why this specific code exists and what it is meant to accomplish. They are local to the code they annotate.
- **Documentation files** explain _process_: the conventions, patterns, and decisions that govern how code is written. They are independent of any specific code location.

If information is only relevant when reading a particular piece of code, it belongs in a comment. If it governs how code is written across the project, it belongs in documentation.

</rules>
