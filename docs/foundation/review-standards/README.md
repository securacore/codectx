# Review Standards

Every documentation update requires a review pass to ensure the content meets the project's foundational standards. This document defines the review checklist. The [docs-audit prompt](../prompts/docs-audit/README.md) automates this review.

Review is not optional. New documentation and updated documentation are both subject to the full checklist. Partial compliance creates inconsistency that degrades AI parsing reliability over time.

## Review Checklist

The following checks apply to all files in `docs/`, excluding `docs/prompts/`.

### Formatting Compliance

Verify against [markdown.md](../markdown/README.md):

- Single H1 per file, always first.
- Maximum heading depth is H3. No H4 or deeper.
- No em dashes in any form (`—`, `--`). Restructure using commas, colons, semicolons, periods, or parentheses.
- Inline code backticks for code references, file names, CLI commands, and configuration keys.
- Fenced code blocks with language identifiers.
- Unordered lists use hyphens.
- No hard line wrapping. Each paragraph is a single continuous line.
- Blank line before and after every heading.
- Blank line before and after every list block.
- Relative paths for internal links.
- No positional references ("see above," "as mentioned earlier"). Use explicit section names or links.

### Linguistic Compliance

Verify against [ai-authoring.md](../ai-authoring/README.md):

- Declarative language. No hedging ("should," "might want to," "consider"). Rules are statements, not suggestions.
- Direct imperatives. "Use X" not "It would be good to use X."
- One instruction per sentence. No compound instructions.
- Positive framing. "Always include X" not "Don't forget to not skip X."
- No negation chains. State the positive rule, then list exceptions.
- Consistent terminology. The same term for the same concept across all files.

### Example Compliance

Verify against [ai-authoring.md](../ai-authoring/README.md):

- Every critical rule that governs concrete patterns (code structure, file organization, naming, syntax) includes at least one positive example (correct) and one negative example (labeled incorrect).
- Negative examples are explicitly labeled as incorrect with a comment or marker.
- Examples are realistic and mirror actual input complexity. No toy examples.
- Complex multi-step rules include 2+ positive and 1+ negative examples.
- Rules that govern abstract guidance (decision-making heuristics, evaluation criteria, architectural principles) do not require forced negative examples. A contrived example that does not genuinely aid comprehension degrades document quality. If a negative example would be contrived, omit it.

### Semantic Marker Compliance

Verify against [markdown.md](../markdown/README.md):

- All constraint sections are wrapped in `<rules>` / `</rules>` markers.
- Execution steps in prompts are wrapped in `<execution>` / `</execution>` markers.
- Semantic markers wrap content inside sections, not around headings.

### Token Positioning

Verify against [ai-authoring.md](../ai-authoring/README.md):

- The most critical rules appear in the first 10% of the document.
- The most critical constraints are repeated or reinforced in the last 10% of the document.
- Supporting detail, examples, and edge cases are in the middle.

### Structural Integrity

Verify against [documentation.md](../documentation/README.md) and [specs.md](../specs/README.md):

- Spec compliance: spec files contain only Purpose, Decisions, Dependencies, Structure sections (in that order). Decisions contain reasoning only, never restated conventions.
- Cross-reference integrity: all relative links resolve to existing files.
- No redundancy: no substantive duplication across files (full sentences or paragraphs conveying the same information in multiple locations).
- Concordance accuracy: if a topic directory uses a concordance `README.md`, the table of links matches the actual files present.

### Integration Verification

- New topic directories are added to the Topics table in [docs/README.md](../README.md).
- New foundational documents are added to the Foundational table in [docs/README.md](../README.md).
- New specs follow the template in [specs.md](../specs/README.md).
- `(future)` references are accurate: they reference topic directories that do not yet exist. References to existing directories use live links.
- Dependencies sections in specs list all documents the topic depends on.
- Structure sections in specs list all files in the topic directory.
