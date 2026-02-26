# CMDX Compression

CMDX (Compressed Markdown Exchange) is a purpose-built text codec that compresses Markdown documentation into a compact, human-readable format optimized for AI token consumption. It is not a general-purpose compression algorithm — it exploits the structural patterns of Markdown and technical documentation specifically.

## Why Not Just Use Markdown?

Markdown is designed for human readability, not token efficiency. It uses verbose structural patterns — repeated heading markers, blockquote prefixes on every line, table alignment padding, redundant emphasis delimiters — that consume tokens without adding semantic value for AI. A technical document with API endpoint tables, parameter lists, and code examples can lose 20-30% of its tokens to structural overhead.

CMDX replaces this overhead with compact `@TAG` syntax while preserving the full semantic content. The output is still a text format that AI models process natively — no special decoding step is needed at inference time.

## Format Overview

CMDX is a line-oriented text format. Every encoded document starts with a version header, an optional dictionary block, and a sequence of tagged content blocks:

```
@CMDX v1
@DICT{
$0=authentication
$1=endpoint
$2=parameter
}
@H1 API Reference
@P This document describes the $0 $1s.
@ENDPOINT{GET /api/users}
@PARAMS{
token:string:R~$0 bearer token
limit:integer:O~Maximum results per page
}
@CODE:json
{"users": [{"id": 1, "name": "Alice"}]}
@/CODE
```

The equivalent Markdown would be substantially longer — a heading, a paragraph, an H3 heading for the endpoint, a 4-column table for parameters, and a fenced code block. CMDX collapses the structural overhead while preserving every piece of information.

## Encoding Pipeline

Encoding runs in four passes:

### Pass 1: Parse

The Markdown source is parsed using goldmark (with GFM extensions for tables, strikethrough, and autolinks) into an internal AST. Each Markdown construct maps to a typed node: `TagH1`–`TagH6` for headings, `TagP` for paragraphs, `TagCodeBlock` for fenced/indented code, `TagUL`/`TagOL` for lists, `TagTable` for GFM tables, and inline tags (`TagBold`, `TagItalic`, `TagCode`, `TagLink`, `TagImage`, `TagStrikethrough`) for inline formatting.

Soft line breaks become spaces. Hard line breaks become explicit `@BR` nodes. Consecutive raw text nodes are merged.

### Pass 2: Domain Pattern Detection

The AST is scanned for common documentation patterns that can be represented more compactly as domain-specific blocks:

- **Key-Value tables** — A 3-column table with headers matching {Field/Key/Name/Parameter, Type, Description} is collapsed into `@KV{key:type~description}` entries, eliminating the table structure overhead entirely.
- **Parameter tables** — A 4-column table with headers {Name, Type, Required, Description} where the Required column contains only "yes"/"no" values is collapsed into `@PARAMS{name:type:R|O~description}`.
- **API endpoints** — An H3 heading matching `METHOD /path` (GET, POST, PUT, DELETE, PATCH) is collapsed into `@ENDPOINT{GET /api/path}`.
- **Admonitions** — Blockquotes starting with bold "Note:", "Warning:", "Tip:", or "Important:" (or GitHub-style `[!NOTE]` callout syntax) are collapsed into `@NOTE{text}`, `@WARN{text}`, `@TIP{text}`, `@IMPORTANT{text}`.

Domain detection is enabled by default and can be disabled via encoder options.

### Pass 3: Dictionary Compression

Repeated strings across the document are identified and replaced with short `$N` references. This is particularly effective for technical documentation where terms like `authentication`, `configuration`, `endpoint`, long URL prefixes, or API parameter names appear many times.

The algorithm:

1. **Candidate extraction.** All text segments (excluding code blocks) are scanned for substrings at word boundaries. Candidates must be at least 10 characters and appear at least twice.

2. **Scoring.** Each candidate is scored by net byte savings: `(frequency - 1) * length - overhead`, where overhead accounts for the dictionary entry line and all `$N` references. Only positive-score candidates provide actual compression.

3. **Greedy selection with overlap tracking.** The highest-scoring candidate is selected, and all character positions it occupies are marked as claimed. Remaining candidates have their effective frequencies recalculated — only unclaimed occurrences count. Scores are recomputed and the process repeats until the dictionary is full (default: 50 entries) or no positive-score candidates remain.

4. **Application.** Selected entries are assigned sequential indices (`$0`, `$1`, ...) ordered by first occurrence in the document. All occurrences in text fields are replaced with `$N` references. Replacements that would create ambiguous references (e.g., `$1` immediately followed by a digit) are skipped.

Code blocks are never dictionary-compressed. The `@` and `$` characters in text are escaped (`@@`, `$$`) before dictionary processing so the codec's own syntax characters are never ambiguous.

### Pass 4: Serialization

The AST is serialized to the CMDX text format. Block-level nodes produce `@TAG content` (single-line) or `@TAG{...}` (multi-line with braces). Inline content is rendered with `@B{bold}`, `@I{italic}`, `@C{code}`, `@LINK{text>url}`, etc. Context-specific escaping is applied to each structural delimiter (pipes in table cells, colons in KV fields, braces everywhere).

The dictionary is written as a `@DICT{...}` block at the top of the file, followed by the body content.

## Tag Reference

### Block Tags

| Tag | Syntax | Markdown Equivalent |
|-----|--------|---------------------|
| `@H1`–`@H6` | `@H1 content` | `# heading` – `###### heading` |
| `@P` | `@P content` | Paragraph |
| `@HR` | `@HR` | `---` |
| `@BQ` | `@BQ{...}` | `> blockquote` |
| `@CODE` | `@CODE:lang` ... `@/CODE` | `` ```lang ... ``` `` |
| `@UL` | `@UL{ - item }` | `- item` |
| `@OL` | `@OL{ 1. item }` | `1. item` |
| `@TABLE` | `@TABLE{ @THEAD{h1\|h2} row }` | GFM table |

### Domain Tags

| Tag | Syntax | Markdown Equivalent |
|-----|--------|---------------------|
| `@KV` | `@KV{ key:type~desc }` | 3-column table (Field/Type/Description) |
| `@PARAMS` | `@PARAMS{ name:type:R\|O~desc }` | 4-column table (Name/Type/Required/Description) |
| `@ENDPOINT` | `@ENDPOINT{METHOD /path}` | `### METHOD /path` |
| `@NOTE` | `@NOTE{text}` | `> **Note:** text` |
| `@WARN` | `@WARN{text}` | `> **Warning:** text` |
| `@TIP` | `@TIP{text}` | `> **Tip:** text` |
| `@IMPORTANT` | `@IMPORTANT{text}` | `> **Important:** text` |

### Inline Tags

| Tag | Syntax | Markdown Equivalent |
|-----|--------|---------------------|
| `@B{...}` | Bold | `**...**` |
| `@I{...}` | Italic | `*...*` |
| `@BI{...}` | Bold+Italic | `***...***` |
| `@C{...}` | Code span | `` `...` `` |
| `@S{...}` | Strikethrough | `~~...~~` |
| `@LINK{text>url}` | Link | `[text](url)` |
| `@IMG{alt>url}` | Image | `![alt](url)` |

## Round-Trip Semantics

CMDX is **semantically lossless**: decoding a CMDX document back to Markdown produces content with the same semantic structure as the original. However, the round-trip is **not byte-identical**. Differences include:

- Emphasis delimiters may normalize (`_emphasis_` becomes `*emphasis*`)
- Fence characters may switch between backticks and tildes
- Blockquote formatting is normalized to a canonical form
- Domain blocks decode to canonical table/heading formats (e.g., KV tables always produce "Field/Type/Description" headers regardless of original header text)
- GitHub-style callout syntax (`[!NOTE]`) decodes to bold-prefix style (`> **Note:** text`)
- Soft line breaks become spaces; trailing whitespace is normalized

This is by design. The codec preserves meaning, not formatting. For AI consumption, semantic equivalence is what matters.

## Code Block Safety

Code blocks receive special treatment throughout the pipeline:

- **Never dictionary-compressed.** Code content is excluded from candidate extraction and reference substitution.
- **Never body-escaped.** The `@@`/`$$` escaping that protects text content is skipped for code blocks.
- **Line-level escaping only.** Lines starting with `@` or `\` are prefixed with `\` to prevent the parser from misinterpreting code as CMDX tags. This is the only transformation applied.
- **Language preserved.** The original language identifier is retained in `@CODE:lang`.

## Compression in the Compile Pipeline

When compression is enabled in preferences (`compression: true`, the default for new projects), the compile step encodes each Markdown file through CMDX before storing it in the object store. Compressed objects use the `.cmdx` file extension instead of `.md`. The content hash used for the filename is computed from the **compressed output**, not the original Markdown source, because the stored content is the compressed form.

Link rewriting happens before compression — the compiler rewrites relative markdown links to content-addressed references first, then encodes the rewritten content to CMDX.

Compression is purely a compile-time optimization. Source files in `docs/` are never modified. The CMDX format is designed so AI models can read it natively without a decoding step.

## Performance Characteristics

- **API documentation**: ~25% byte/token savings (heavy table structure, repeated terms)
- **Prose-heavy content**: ~10-15% savings (less structural overhead, fewer repeated terms)
- **Code-heavy documents**: Minimal savings (code blocks are preserved verbatim)
- **Deterministic output**: Same input always produces the same encoded output
- **Fast encoding**: Sub-millisecond for typical documentation files

## CLI Tools

The `codectx cmdx` subcommand provides utilities for working with CMDX directly:

- `codectx cmdx stats <file>` — Show compression statistics (original/compressed bytes, savings, dict entries)
- `codectx cmdx validate <file>` — Encode and decode, verify semantic round-trip
- `codectx cmdx roundtrip <file>` — Encode to CMDX and decode back to Markdown

## Related

- [Compilation](compilation.md) — how compression integrates with the compile pipeline
- [Configuration](configuration.md) — how to enable/disable compression
- [Preference Management](set-command.md) — `codectx set compression=true|false`
- [Design Decisions](spec/README.md) — reasoning behind CMDX design choices
