# Markdown

Markdown conventions for documentation files in this project. These govern files within `docs/` and any documentation-specific markdown.

These conventions are chosen with AI consumption as a primary concern. Structure, consistency, and predictability aid both human readers and AI agents that parse these files as instructions.

## Writing for AI

These guidelines improve how AI agents parse, navigate, and act on documentation.

<rules>

- **Descriptive heading text.** Headings serve as semantic anchors for AI navigation. A heading like "Configuration" is ambiguous across documents; "Database Configuration" is directly navigable. Write headings that can stand alone without surrounding context.
- **No positional references.** Avoid phrases like "see above," "as mentioned earlier," or "the following section." AI may load documents partially, out of order, or in isolation. Use explicit section names or inline links to the referenced location.
- **Front-load intent.** The introductory paragraph after H1 and the opening sentence of each section states what the section covers. AI uses these to decide whether to read deeper or skip ahead.
- **Consistent terminology.** Use the same term for the same concept throughout all documentation. Alternating between synonyms degrades AI pattern-matching and search reliability. Pick one term and use it everywhere.

</rules>

For linguistic patterns and cross-model authoring conventions, see [ai-authoring](../ai-authoring/README.md).

### Semantic Boundary Markers

Semantic boundary markers are paired HTML tags used to classify sections of a document by intent. They tell an AI agent _how_ to treat a section (as steps to execute, constraints to observe, context to absorb) rather than relying on heading text alone.

Use lowercase, descriptive tag names that reflect the section's role:

- `<execution>` / `</execution>`: content the AI carries out as sequential steps
- `<rules>` / `</rules>`: constraints the AI must observe during execution
- `<context>` / `</context>`: background information the AI absorbs but does not act on directly

Semantic markers wrap content _inside_ a section, not around headings. Place the opening tag after the heading and any introductory prose. Place the closing tag before the next heading or at the end of the section.

These tags are not rendered visually in standard markdown viewers. They exist solely for AI parsing. Do not remove them during formatting cleanup.

## File Conventions

- **Naming:** `kebab-case.md` for documentation files. The only exception is `README.md`, which uses the universally recognized uppercase form.
- **Formatting:** UTF-8, LF line endings, trimmed trailing whitespace, final newline required.

## Document Structure

Every documentation file follows this structure:

- A single H1 (`#`) serves as the document title. One per file, always first.
- An optional introductory paragraph immediately after H1, describing the document's scope.
- H2 (`##`) for major sections.
- H3 (`###`) for subsections within an H2.
- **Maximum depth is H3.** If content requires H4 or deeper, restructure the document or split it into separate files.
- A blank line must appear before and after every heading.

## Prose

- No hard line wrapping. Write each paragraph as a single continuous line. Let the editor or viewer handle soft wrapping.
- Separate paragraphs with a blank line.
- Keep paragraphs concise. Break long paragraphs into multiple paragraphs or a list.

## Inline Formatting

- **Emphasis (italic):** Use underscores: `_text_`.
- **Strong emphasis (bold):** Use double asterisks: `**text**`.
- **Inline code:** Use backticks: `` `text` `` for code references, file names, CLI commands, and configuration keys.

## Links

- Use inline links: `[text](url)`.
- Use relative paths for links to other files within the project.
- Avoid reference-style links.

## Lists

- **Unordered lists:** Use hyphens (`-`).
- **Ordered lists:** Use incrementing numbers (`1.`, `2.`, `3.`).
- **Nesting:** Indent nested items by 2 spaces.
- Prefer prose over a list when there are only one or two items.
- A blank line before and after a list block.

## Code Blocks

- Use fenced code blocks with triple backticks.
- Always include a language identifier. Use `text` for content that is not a recognized language.

## Tables

- Tables are permitted for structured or comparative data.
- Left-align columns by default.
- Avoid tables for simple key-value information. Use a list instead.

## Raw HTML

Raw HTML is prohibited in documentation files for layout or formatting purposes.

The one exception is semantic boundary markers, as described in the Semantic Boundary Markers section. These tags serve AI parsing, not visual rendering, and are permitted.
