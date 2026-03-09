# Specification Documentation

This document defines the format and process for writing specification documents. Specs are meta-documentation. They capture the reasoning and decisions behind _why_ documentation was created the way it was, so it can be understood, recreated, or revised in the future.

## Location

Every topic directory may contain a `spec/` subdirectory with a `README.md` entry point. For complex specs, `README.md` links to child files within the same `spec/` directory.

- Topic specs: `docs/topics/[topic]/spec/README.md`
- Foundational spec: `docs/foundation/spec/README.md`

## Template

Every spec uses the structure defined in this section. All sections are optional. Include only what is relevant. When a section is present, it must follow the format described here. Sections appear in the order listed.

### Purpose

Why this documentation exists. One to three sentences. State the problem it solves or the gap it fills.

### Decisions

Key choices made during creation. Each decision is an H3 entry (when the spec is a standalone `README.md`) or a list item (when brevity is preferred). The bold label _names_ the decision as an identifier. The body contains _only reasoning_: why the choice was made and what alternatives were rejected. Do not restate the convention itself; the conventions document is the authoritative source for what the rule is. The spec records why that rule exists.

Format for list-style decisions:

- **Decision label.** Why this choice was made. Alternative considered: alternative A (why rejected).

Format for heading-style decisions (use when a decision requires more than two sentences of reasoning):

```text
### Decision Label

Why this choice was made. What alternatives were considered and why they were rejected.
```

### Dependencies

What this documentation relies on: other documentation files, configuration files, external tools. Plain list of references with brief context for each.

### Structure

How the documentation within this topic directory is organized. List the files that exist and their purpose. This section is a map. It tells a reader (human or AI) where to find what without reading every file.

## Principles

- **Token density.** Specs are concise. No padding, no ceremony, no restating what is obvious from context. Every sentence carries information.
- **Reasoning over description.** Specs record _why_, not _what_. The documentation itself records what. The spec records why the documentation exists and why it was structured that way.
- **Reproducibility.** A spec contains enough reasoning that the documentation could be recreated from scratch by someone (or an AI) who has never seen it, arriving at a substantially similar result.
