# Markdown Specification

Spec for the markdown foundation document.

## Purpose

Consistent markdown formatting across all documentation files ensures AI agents can reliably parse, navigate, and act on the content. Without explicit conventions, each author makes independent formatting decisions (heading depths, list styles, link formats) that create inconsistency. AI models are sensitive to structural variation; predictable formatting reduces parsing errors and improves instruction compliance.

## Decisions

- **H3 maximum depth.** Deeper heading levels (H4+) create navigation complexity that degrades AI parsing reliability. When content needs more depth, restructure or split the document. Alternative considered: allowing H4 (rejected; H4+ headings are rarely navigated correctly by AI agents and indicate the document should be split).

- **Semantic boundary markers over heading conventions.** Paired HTML tags (`<rules>`, `<execution>`, `<context>`) classify sections by intent independently of heading text. This is more reliable than relying on heading naming conventions because the tags are machine-parseable and unambiguous. Alternative considered: heading prefixes like "Rules:" (rejected; mixing metadata into heading text reduces heading clarity and is less reliably parsed).

- **No hard line wrapping.** Each paragraph is a single continuous line. This eliminates diff noise from rewrapping and ensures AI processes each paragraph as a single unit. Alternative considered: 80-character wrapping (rejected; creates artificial line breaks within sentences that AI may interpret as separate items).

- **Hyphens for unordered lists.** Standardizing on hyphens (not asterisks or plus signs) eliminates formatter disagreements and creates visual consistency. All three are valid markdown; picking one and enforcing it prevents mixed styles.

- **No raw HTML except semantic markers.** Raw HTML for layout breaks the markdown abstraction and creates rendering inconsistencies. Semantic markers are the sole exception because they serve AI parsing, not visual rendering.

- **`load: documentation` for this document.** Markdown conventions are needed when writing documentation, not when writing code. Loading them in code-only sessions wastes tokens. Alternative considered: `load: always` (rejected; code sessions do not create markdown files).

## Dependencies

- [ai-authoring](../../ai-authoring/README.md): linguistic conventions that complement formatting conventions
