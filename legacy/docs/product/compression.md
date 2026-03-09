# Markdown Compression

The compression system reduces token consumption when feeding documentation to LLMs. It takes human-written Markdown and compiles it into compact, normalized Markdown that is maximally token-efficient for BPE tokenization. The output is valid, native Markdown that AI models read directly without any decoding step.

## Why Compress Markdown?

Markdown written for humans contains structural overhead that costs tokens without adding value for AI. Soft-wrapped lines create unnecessary whitespace tokens. Table alignment padding adds bytes. Repeated URLs waste tokens on every occurrence. TOC-style navigation tables carry pipe delimiters and header rows that serve formatting, not content.

The compression pipeline eliminates this overhead while preserving all semantic content. The output is still Markdown. Any Markdown parser reads it correctly. The difference is that every byte is chosen for token efficiency.

## Encoding Pipeline

Encoding runs in two passes.

### Pass 1: Parse

The Markdown source is parsed using goldmark (with GFM extensions for tables, strikethrough, and autolinks) into an internal AST. Each Markdown construct maps to a typed node: `TagH1` through `TagH6` for headings, `TagP` for paragraphs, `TagCodeBlock` for fenced and indented code, `TagUL`/`TagOL` for lists, `TagTable` for GFM tables, and inline tags (`TagBold`, `TagItalic`, `TagCode`, `TagLink`, `TagImage`, `TagStrikethrough`) for inline formatting.

Soft line breaks become spaces. Hard line breaks become explicit break nodes. Consecutive raw text nodes are merged. TOC-style tables (2-column tables where column 1 is entirely a link) are detected and converted to bullet lists, preserving the link URLs that would otherwise be lost during table cell text extraction.

### Pass 2: Serialize

The AST is serialized to compact Markdown with these BPE-optimized normalizations applied:

- **Soft-wrap joining.** Multi-line paragraphs become single lines. Blank lines between blocks cost zero extra tokens in o200k_base encoding, so block separation is preserved.
- **Compact table format.** Tables use `|cell|cell|` with no spaces around pipes and a minimal `|-|-|` separator row. This saves approximately 6% of table tokens compared to spaced format.
- **TOC table-to-list conversion.** Two-column tables where every row has a link in column 1 are converted to `- [text](url) -- description` bullet lists. This preserves navigation URLs and eliminates header and separator row overhead.
- **Reference-style link deduplication.** URLs appearing two or more times are evaluated against the o200k_base tokenizer. When reference-style links (`[text][1]` with `[1]: URL` definitions) cost fewer tokens than repeated inline links, the encoder uses reference style. Short URLs that would cost more as references stay inline. The decision is data-driven per URL.
- **Table cell escaping.** Pipe characters and trailing backslashes in compact table cells are escaped to prevent misparse.
- **Inline escaping.** Markdown-significant characters (`*`, `_`) in literal text are backslash-escaped to prevent formatting reinterpretation during re-parse.

## Round-Trip Semantics

Compression is **semantically lossless**: re-parsing the compressed output with goldmark produces the same AST structure as the original input. The round-trip is **not byte-identical**. Differences include:

- Soft line breaks become spaces (joined paragraphs)
- Table alignment markers are dropped (AI understands tables from headers and content)
- Link and image titles are dropped (rarely used, zero semantic value for AI)
- Multiple consecutive blank lines collapse to single
- Trailing whitespace is normalized
- TOC tables become bullet lists (links preserved, table headers dropped)

Semantic equivalence is validated by comparing goldmark AST dumps of the original and compressed documents. The comparison strips source positions, table alignment, and link titles before diffing. An extensive fuzz test corpus exercises edge cases in escaping, emphasis nesting, strikethrough adjacency, hard breaks, and list item boundaries.

## Code Block Safety

Code blocks are preserved verbatim throughout the pipeline. No normalization, escaping, or transformation is applied to code block content. The language identifier is retained. Fence characters (backticks or tildes) and fence length are chosen to avoid conflicts with code content.

## Compression in the Compile Pipeline

When compression is enabled in preferences (`compression: true`, the default for new projects), the compile step encodes each Markdown file through the compression pipeline before storing it in the object store. Compressed objects use the `.ctx.md` file extension. The content hash used for the filename is computed from the original Markdown source, not the compressed output.

Link rewriting happens before compression. The compiler rewrites relative Markdown links to content-addressed references first, then encodes the rewritten content. Compressed objects contain links to content-addressed hashes, not to original source paths.

Compression is purely a compile-time optimization. Source files in `docs/` are never modified.

## Performance

Real corpus measurement across 55 documentation files (430 KB total):

- **Aggregate:** 95,279 raw tokens compressed to 93,096 tokens (2.3% reduction, 2,183 tokens saved)
- **Index pages with TOC tables:** 10-25% token savings (heavy structural overhead in original)
- **Structured content with tables:** 3-7% savings (compact table format eliminates padding)
- **Prose-heavy content:** 0% savings (already information-dense, minimal structural waste)
- **Code-heavy documents:** 0% savings (code blocks are preserved verbatim)

The tokenizer uses o200k_base encoding (GPT-4o class), matching the baseline model assumption defined in [ai-authoring](../foundation/ai-authoring/README.md).

## CLI Tools

The `codectx md` subcommand provides utilities for working with compression directly:

- `codectx md encode <file>` — Compress a Markdown file and write the output to stdout or a file
- `codectx md stats <file>` — Show compression statistics (original and compressed bytes and tokens, savings percentage). Supports `--dir` for corpus-wide measurement.
- `codectx md roundtrip <file>` — Encode and re-parse, verify semantic equivalence

## Related

- [Compilation](compilation.md) — how compression integrates with the compile pipeline
- [Configuration](configuration.md) — how to enable or disable compression
- [Preference Management](set-command.md) — `codectx set compression=true|false`
- [Design Decisions](spec/README.md) — reasoning behind architectural choices
