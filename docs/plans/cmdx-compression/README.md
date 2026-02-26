# CMDX Compression — Implementation Plan

CMDX (Compressed Markdown Exchange) is a lossless, text-based compression codec for Markdown that reduces token consumption when feeding documentation to LLMs. It lives inside the codectx CLI as `core/cmdx/` and produces compressed output that AI reads directly — no decompression step at inference time.

This plan supersedes `docs/plans/lossless-markdown-compression/`, which contains the original specification and sample I/O. That directory remains as reference material. This document is the authoritative implementation plan.

---

## Goals

1. Build a Go library (`core/cmdx/`) that encodes Markdown to CMDX and decodes CMDX back to Markdown.
2. Expose it as a CLI subcommand: `codectx cmdx encode|decode|stats|validate|roundtrip`.
3. Achieve 20–35% token reduction on structured technical documentation (API docs, codectx foundation docs, topic docs).
4. Guarantee content-lossless round-tripping: all textual content survives encode→decode unchanged.
5. Produce output that is directly readable by LLMs without decompression.
6. Build an exhaustive test suite covering every feature, edge case, and invariant.

## Non-Goals

- Human-readable compressed output. The format is optimized for AI consumption.
- Replacing general-purpose compression (gzip, zstd) for storage or bandwidth.
- Lossy summarization or content reduction.
- Compile pipeline integration (planned separately after library validation).
- Compressing prose-heavy content with minimal structural repetition (savings will be low and that is acceptable).

---

## Design Decisions

Every decision made during planning is recorded here with rationale.

### D1: Content-lossless, presentation-lossy

The round-trip guarantee is "all textual content survives." Presentational metadata that carries no semantic value for AI is intentionally dropped:

| Dropped | Rationale |
|---------|-----------|
| Table column alignment (`:---:`, `:---`, `---:`) | AI understands tables from headers + content, not alignment markers |
| Link/image titles (`[text](url "title")`) | Rarely used in technical docs, zero semantic value for AI |
| Reference-style → inline link conversion | goldmark resolves references before AST construction; same content, different syntax |
| Multiple consecutive blank lines → single | Whitespace is presentational, not semantic |
| Trailing whitespace on lines | Never meaningful outside code blocks |

The round-trip comparison function must strip these attributes from both ASTs before comparing.

### D2: AST comparison for round-trip validation

Round-trip correctness is validated by comparing goldmark AST trees, not markdown text. Both the original input and the decoded output are parsed into goldmark ASTs, serialized to a canonical tree dump (node type + semantic attributes, no source positions), and compared as strings. This provides:

- Precise semantic comparison (not affected by whitespace normalization)
- Readable diff output when tests fail (shows exactly which node diverged)
- No need for a canonical Markdown renderer (which would be a project unto itself)

The canonical tree dump function strips source positions, table alignment attributes, and link/image title attributes before serialization.

### D3: Escape `@` at line start inside code blocks

If a line inside a code block starts with `@`, the encoder prefixes it with `\`: the line `@/CODE` becomes `\@/CODE`. The decoder strips the leading `\` from lines inside code blocks that start with `\@`. This prevents the parser from interpreting source code containing `@/CODE` as a block terminator.

This is the only escaping rule specific to code blocks. All other escaping rules apply to body text only.

### D4: First unescaped `>` splits LINK/IMG display from URL

`@LINK{display text>https://example.com}` — the parser splits on the first `>` not preceded by `\`. Everything before is display text (with `\>` unescaped to `>`), everything after is the URL (literal, no escaping needed). Since URLs are the tail segment, `>` characters in URLs do not cause ambiguity.

### D5: Always escape `$` to `$$`

Every literal `$` in body text is encoded as `$$`, regardless of what follows. `$PATH` becomes `$$PATH`. `$5` becomes `$$5`. This is one rule with zero ambiguity. The parser never has to inspect the character after `$` to decide if it is a reference or literal.

### D6: Escape ordering

**Encoder (markdown → CMDX):**

1. Escape literal `@` → `@@` and `$` → `$$` in source text content FIRST
2. THEN replace dictionary-matched substrings with `$N` references

If this order is reversed, a source text containing literal `$5` would collide with dictionary reference `$5`.

**Decoder (CMDX → markdown):**

1. Expand `$N` dictionary references FIRST
2. THEN unescape `@@` → `@` and `$$` → `$`

If this order is reversed, a dictionary value containing `@@` would be incorrectly unescaped.

### D7: Dictionary expansion walks ALL string fields

The decoder's dictionary expansion pass must walk every string field in every AST node, not just `Node.Content`. Fields that can contain `$N` references:

- `Node.Content` (paragraph text, heading text, blockquote text)
- `NodeAttrs.URL` (link and image URLs)
- `NodeAttrs.Display` (link and image display text)
- `NodeAttrs.Callout` (admonition content)
- `KVItem.Key`, `KVItem.Type`, `KVItem.Description`
- `ParamItem.Name`, `ParamItem.Type`, `ParamItem.Description`
- `ReturnDef.Status`, `ReturnDef.Description`
- Table cell values (`NodeAttrs.Cells[][]`, `NodeAttrs.Headers[]`)

Missing a field means silent data loss on round-trip. The implementation must have a test that verifies dictionary references work in every field type.

### D8: goldmark GFM extensions required

All goldmark parser instances must enable the GFM extension:

```go
goldmark.New(goldmark.WithExtensions(extension.GFM))
```

This enables tables, strikethrough, and task lists. Both the encoder (markdown → AST) and the comparison function (decoded markdown → AST for round-trip validation) must use identical goldmark configurations. Divergent configurations would produce different ASTs for the same input, breaking the round-trip invariant.

### D9: Paragraph continuation lines are joined

Markdown soft line breaks (single newlines within a paragraph) are joined into a single `@P` line. Hard line breaks (two trailing spaces or `\` at end of line, or `<br>`) are represented as `@BR` within the paragraph. This is correct because soft line breaks render as spaces in HTML and carry no semantic meaning.

### D10: Location is `core/cmdx/` inside codectx

CMDX lives as a package within the codectx module at `github.com/securacore/codectx/core/cmdx`. It shares the module's `go.mod` and adds `github.com/yuin/goldmark` as a dependency. The CLI subcommand is at `cmds/cmdx/`.

### D11: Compile integration is deferred

The compile pipeline (`core/compile/`) stores markdown as content-addressed objects in `.codectx/objects/`. CMDX integration with compile is a separate effort planned after the standalone library is validated with real documentation. The initial deliverable is the library + CLI subcommand only.

---

## File Map

### New files to create

```
core/cmdx/
├── Encode.go              # Public Encode() function + encoder pipeline
├── Decode.go              # Public Decode() function + decoder pipeline
├── AST.go                 # Document, Node, NodeAttrs, TagType, all domain types
├── Dict.go                # Dictionary, DictEntry, BuildDictionary()
├── Tags.go                # TagParser, tag definitions, parsing, serialization
├── Escape.go              # EscapeBody, UnescapeBody, EscapeCell, etc.
├── Normalize.go           # AST comparison: DumpAST(), CompareASTs()
├── Stats.go               # Stats struct, Analyze() function
├── Encode_test.go         # Encoder-specific tests (per-tag, per-feature)
├── Decode_test.go         # Decoder-specific tests (per-tag, per-feature)
├── Dict_test.go           # Dictionary builder tests
├── Tags_test.go           # Tag parser tests
├── Escape_test.go         # Escape/unescape round-trip tests
├── Normalize_test.go      # AST dump and comparison tests
├── Roundtrip_test.go      # Round-trip invariant tests (the critical suite)
├── Stats_test.go          # Stats calculation tests
├── Fuzz_test.go           # Fuzz tests for round-trip safety
└── testdata/
    ├── simple.md          # Basic syntax coverage fixture
    ├── api_docs.md        # Complex API docs fixture (from original plan)
    ├── api_docs.cmdx      # Expected compressed output
    ├── edge_cases.md      # Edge case fixture (escaping, nesting, code blocks)
    ├── prose.md           # Prose-heavy fixture (low compression expected)
    └── empty.md           # Empty document fixture

cmds/cmdx/
├── main.go                # Parent command: codectx cmdx
├── encode.go              # Subcommand: codectx cmdx encode
├── decode.go              # Subcommand: codectx cmdx decode
├── stats.go               # Subcommand: codectx cmdx stats
├── validate.go            # Subcommand: codectx cmdx validate
├── roundtrip.go           # Subcommand: codectx cmdx roundtrip
└── main_test.go           # Command registration and error guard tests
```

### Modified files

```
main.go                    # Register cmds/cmdx command
go.mod                     # Add github.com/yuin/goldmark dependency
go.sum                     # Updated by go mod tidy
```

### Reference files (read, not modified)

```
docs/plans/lossless-markdown-compression/CMDX_SPECIFICATION.md   # Original spec
docs/plans/lossless-markdown-compression/api_docs.md             # Sample input
docs/plans/lossless-markdown-compression/api_docs.cmdx           # Expected output
docs/plans/lossless-markdown-compression/simple.md               # Basic fixture
```

---

## Type Definitions

All types live in `core/cmdx/AST.go`. These are taken from the original spec with minor refinements.

```go
package cmdx

// Document represents a parsed CMDX document.
type Document struct {
    Version  string
    Dict     *Dictionary
    Meta     map[string]string
    Body     []Node
}

// Dictionary maps $N reference tokens to expanded string values.
type Dictionary struct {
    Entries []DictEntry
    index   map[string]string // reverse lookup: value → "$N" (encoder use)
}

type DictEntry struct {
    Index int
    Value string
}

// Node represents a single element in the document AST.
type Node struct {
    Tag      TagType
    Content  string
    Attrs    NodeAttrs
    Children []Node
}

type NodeAttrs struct {
    Language string       // CODE blocks
    Level    int          // Headings (1-6)
    Items    []KVItem     // KV blocks
    Params   []ParamItem  // PARAMS blocks
    Cells    [][]string   // TABLE body rows
    Headers  []string     // TABLE header row
    URL      string       // LINK, IMG
    Display  string       // LINK, IMG
    Callout  string       // NOTE, WARN, TIP, IMPORTANT
    Endpoint *EndpointDef // ENDPOINT blocks
    Returns  []ReturnDef  // RETURNS blocks
}

type KVItem struct {
    Key         string
    Type        string
    Description string
}

type ParamItem struct {
    Name        string
    Type        string
    Required    bool   // R=true, O=false
    Description string
}

type EndpointDef struct {
    Method string // GET, POST, PUT, DELETE, PATCH
    Path   string
}

type ReturnDef struct {
    Status      string
    Description string
}

type TagType int

const (
    TagH1 TagType = iota
    TagH2
    TagH3
    TagH4
    TagH5
    TagH6
    TagHR
    TagBR
    TagP
    TagBQ
    TagBold
    TagItalic
    TagBoldItalic
    TagCode
    TagCodeBlock
    TagStrikethrough
    TagLink
    TagImage
    TagUL
    TagOL
    TagTable
    TagKV
    TagParams
    TagEndpoint
    TagReturns
    TagDef
    TagNote
    TagWarn
    TagTip
    TagImportant
    TagMeta
    TagDict
    TagRaw
)
```

## Public API

```go
package cmdx

// Encode converts Markdown to CMDX format.
func Encode(markdown []byte, opts ...EncoderOptions) ([]byte, error)

// Decode converts CMDX back to Markdown.
func Decode(cmdx []byte) ([]byte, error)

// Parse parses CMDX into a Document AST without decoding to Markdown.
func Parse(cmdx []byte) (*Document, error)

// Analyze returns compression statistics comparing original and compressed.
func Analyze(markdown []byte, cmdx []byte) Stats

// CompareASTs parses two markdown documents and returns whether they are
// semantically equivalent (ignoring presentational metadata).
func CompareASTs(a, b []byte) (bool, string, error)

// EncoderOptions configures the encoding process.
type EncoderOptions struct {
    MaxDictEntries     int  // Default: 50
    MinStringLength    int  // Default: 10
    MinFrequency       int  // Default: 2
    EnableDomainBlocks bool // Default: true
    PreserveMeta       bool // Default: true
}

func DefaultEncoderOptions() EncoderOptions

// Stats holds compression analysis results.
type Stats struct {
    OriginalBytes   int
    CompressedBytes int
    ByteSavings     float64 // percentage
    DictEntries     int
    DictSavings     int     // bytes saved by dictionary alone
    DomainSavings   int     // bytes saved by domain blocks alone
    EstTokensBefore int     // rough estimate: bytes / 4
    EstTokensAfter  int
    TokenSavings    float64 // percentage
}
```

---

## Encoder Pipeline

Four passes, executed in this order. The order is critical and must not change.

### Pass 1: Markdown → Internal AST

Parse input markdown with goldmark (GFM extensions enabled). Walk the goldmark AST and convert each node to the internal `Node` representation.

**goldmark node → internal node mapping:**

| goldmark Node | Internal Tag | Notes |
|---------------|-------------|-------|
| `ast.Heading` (level 1-6) | `TagH1`–`TagH6` | |
| `ast.Paragraph` | `TagP` | Join soft-wrapped lines into single content string |
| `ast.ThematicBreak` | `TagHR` | |
| `ast.Blockquote` | `TagBQ` | Recurse into children for multi-line |
| `ast.FencedCodeBlock` | `TagCodeBlock` | Preserve language, preserve content exactly |
| `ast.CodeBlock` | `TagCodeBlock` | Indented code blocks (no language) |
| `ast.List` (ordered) | `TagOL` | Recurse into ListItems |
| `ast.List` (unordered) | `TagUL` | Recurse into ListItems |
| `ast.ListItem` | Child of UL/OL | |
| `extast.Table` | `TagTable` | Extract headers and cell values |
| `ast.Text` | `TagRaw` | Leaf text content |
| `ast.CodeSpan` | `TagCode` | Inline code |
| `ast.Emphasis` (level 1) | `TagItalic` | |
| `ast.Emphasis` (level 2) | `TagBold` | |
| `ast.Emphasis` (level 3) | `TagBoldItalic` | If goldmark reports level 3 |
| `ast.Link` | `TagLink` | Extract URL and display text |
| `ast.Image` | `TagImage` | Extract URL and alt text |
| `extast.Strikethrough` | `TagStrikethrough` | |
| `ast.HTMLBlock` | `TagRaw` | Pass through as-is (edge case) |
| `ast.RawHTML` | `TagRaw` | Inline HTML pass-through |

### Pass 2: Detect Domain Patterns

Scan the internal AST for tables that match domain-specific patterns. Replace matched tables with compact domain nodes.

**Detection rules (deterministic, not heuristic):**

| Pattern | Trigger | Result |
|---------|---------|--------|
| KV table | Table headers contain columns matching {`Field`/`Key`/`Name`}, {`Type`}, {`Description`} (case-insensitive, order-independent) AND no `Required` column | Replace with `TagKV` node |
| PARAMS table | Table headers contain columns matching {`Name`/`Parameter`}, {`Type`}, {`Required`}, {`Description`} (case-insensitive, order-independent) | Replace with `TagParams` node. Map `Yes`/`True`/`Y` → `R`, everything else → `O` |
| ENDPOINT heading | H3 text matching `METHOD /path` where METHOD is `GET`/`POST`/`PUT`/`DELETE`/`PATCH` | Replace with `TagEndpoint` node |

**Important:** Detection happens on the AST after Pass 1, before dictionary building. This is because collapsing tables changes the text content the dictionary analyzer sees. Building the dictionary first would create entries for strings that only appeared in the verbose table format.

**Tables that don't match any pattern are left as `TagTable` nodes and serialized with `@TABLE{}`.**

### Pass 3: Build Dictionary

Extract all text content from the AST (excluding code block content), run frequency analysis, build dictionary entries, replace occurrences with `$N` references.

**Algorithm:**

1. Collect all text segments from AST nodes (walk `Content`, `Display`, `Description`, all string fields in `NodeAttrs`). Skip `TagCodeBlock` content entirely.
2. Extract candidate substrings using sliding window at word boundaries, length `MinStringLength` to 100 characters.
3. Count frequency of each candidate across all segments.
4. Filter: keep candidates with `frequency >= MinFrequency` and `length >= MinStringLength`.
5. Score each candidate: `score = (freq - 1) * len(candidate) - overhead` where `overhead = len("$N=candidate\n") + freq * len("$N")`.
6. Sort by score descending. Break ties by first occurrence position (stable sort).
7. Greedy selection: take highest-scoring candidate, mark all its occurrences as claimed. Recalculate scores for remaining candidates (reduce frequency by overlap count). Repeat.
8. Stop when no candidate has positive score OR `MaxDictEntries` reached.
9. Assign final indices 0..N in order of first appearance in the document.

**Determinism requirement:** Same input + same options → same dictionary, every time. Use `sort.SliceStable`. Break all ties by first-occurrence position.

**After dictionary is built:** Walk the AST and replace all occurrences of dictionary values with `$N` references in every string field (same field list as D7). Escaping of `$` to `$$` happens BEFORE this replacement (see D6).

### Pass 4: Serialize to CMDX

Emit the CMDX text output:

1. Emit `@CMDX v1\n`
2. If dictionary is non-empty, emit `@DICT{...}\n`
3. If metadata exists, emit `@META{...}\n`
4. Walk the AST and emit body tags
5. Apply code block escaping: if a line inside `@CODE`/`@/CODE` starts with `@`, prefix with `\`

**Serialization rules per tag type:**

| Tag | Output | Notes |
|-----|--------|-------|
| `TagH1`–`TagH6` | `@H1 text` ... `@H6 text` | One line |
| `TagHR` | `@HR` | One line |
| `TagBR` | `@BR` | Inline |
| `TagP` | `@P text` | One line, continuation joined |
| `TagBQ` | `@BQ text` or `@BQ{...}` | Single-line or multi-line |
| `TagBold` | `@B{text}` | Inline |
| `TagItalic` | `@I{text}` | Inline |
| `TagBoldItalic` | `@BI{text}` | Inline |
| `TagCode` | `@C{text}` | Inline |
| `TagStrikethrough` | `@S{text}` | Inline |
| `TagLink` | `@LINK{display>url}` | Inline |
| `TagImage` | `@IMG{alt>url}` | Inline |
| `TagCodeBlock` | `@CODE:lang\n...\n@/CODE` | Preserve content exactly, escape `@` at line start |
| `TagUL` | `@UL{\n- item\n...}` | Nested lists use indentation + inner `@UL{}`/`@OL{}` |
| `TagOL` | `@OL{\n1. item\n...}` | |
| `TagTable` | `@TABLE{\n@THEAD{col\|col}\nval\|val\n...}` | Standard tables |
| `TagKV` | `@KV{\n  key:type~desc\n...}` | |
| `TagParams` | `@PARAMS{\n  name:type:R\|O~desc\n...}` | |
| `TagEndpoint` | `@ENDPOINT{METHOD /path}` | |
| `TagReturns` | `@RETURNS{status:desc\|status:desc}` | |
| `TagDef` | `@DEF{\n  term~definition\n...}` | |
| `TagNote` | `@NOTE{text}` | |
| `TagWarn` | `@WARN{text}` | |
| `TagTip` | `@TIP{text}` | |
| `TagImportant` | `@IMPORTANT{text}` | |
| `TagMeta` | `@META{key:val;key:val}` | |

---

## Decoder Pipeline

Four passes, the reverse of encoding.

### Pass 1: Parse CMDX → Internal AST

1. Read first line, validate `@CMDX v1` header.
2. Parse `@DICT{...}` block if present → build `Dictionary`.
3. Parse `@META{...}` if present → populate `Meta` map.
4. Parse body using `TagParser`: dispatch on `@TAG` prefixes, handle brace matching for block tags, handle `@CODE`/`@/CODE` blocks as literal content.

**Tag parser rules:**

- Line-level tags (`@H1`, `@HR`, `@BR`, `@P`): tag + content to EOL.
- Inline tags (`@B{}`, `@C{}`, `@LINK{}`, `@IMG{}`, `@S{}`): tag + braced content, can nest.
- Block tags (`@UL{}`, `@OL{}`, `@TABLE{}`, `@KV{}`, `@PARAMS{}`, `@DEF{}`, `@BQ{}`): tag + multi-line braced content.
- Code blocks (`@CODE:lang` ... `@/CODE`): literal content between delimiters, no tag processing inside. Unescape `\@` at line start.
- Single-line tags (`@NOTE{}`, `@WARN{}`, `@TIP{}`, `@IMPORTANT{}`, `@ENDPOINT{}`, `@RETURNS{}`): tag + braced content on one line.
- `@@` in body text is literal `@` (not a tag).
- `$$` in body text is literal `$` (not a reference).

### Pass 2: Expand Dictionary References

Walk every string field in every AST node (same field list as D7). Replace `$N` with `Dictionary.Entries[N].Value`. Process in reverse index order (expand `$49` before `$4`) to avoid prefix collisions, OR use regex/token-based replacement that matches `$` followed by one or more digits as a complete token.

**After expansion:** Unescape `@@` → `@` and `$$` → `$` in all string fields.

### Pass 3: Convert Domain Blocks to Standard Structures

| Domain Tag | Markdown Output |
|------------|----------------|
| `TagKV` | Markdown table with headers: `Field`, `Type`, `Description` |
| `TagParams` | Markdown table with headers: `Name`, `Type`, `Required`, `Description`. `R` → `Yes`, `O` → `No` |
| `TagEndpoint` | H3 heading: `### METHOD /path` |
| `TagReturns` | Inline text or table depending on entry count |
| `TagDef` | Markdown table with headers: `Term`, `Definition` |

### Pass 4: Serialize to Markdown

Walk the AST and emit standard Markdown:

| Internal Tag | Markdown Output |
|-------------|----------------|
| `TagH1`–`TagH6` | `# text` ... `###### text` |
| `TagHR` | `---` |
| `TagBR` | Hard line break (two trailing spaces + newline) |
| `TagP` | Paragraph text + blank line |
| `TagBQ` | `> text` (each line prefixed with `> `) |
| `TagBold` | `**text**` |
| `TagItalic` | `*text*` |
| `TagBoldItalic` | `***text***` |
| `TagCode` | `` `text` `` |
| `TagStrikethrough` | `~~text~~` |
| `TagLink` | `[display](url)` |
| `TagImage` | `![alt](url)` |
| `TagCodeBlock` | `` ```lang\ncontent\n``` `` |
| `TagUL` | `- item` per line, 2-space indent for nesting |
| `TagOL` | `1. item` per line |
| `TagTable` | Standard markdown table with `\|` separators and `---` divider |
| `TagNote` | `> **Note:** text` |
| `TagWarn` | `> **Note:** text` (GitHub-style callout) |
| `TagTip` | `> **Tip:** text` |
| `TagImportant` | `> **Important:** text` |

**Whitespace rules:** Emit one blank line between block-level elements. No trailing whitespace on any line. No multiple consecutive blank lines.

---

## Escaping Rules (Complete Reference)

| Literal Character | Escape Sequence | Context | Applied By |
|---|---|---|---|
| `@` | `@@` | Body text | Encoder Pass 4 / Decoder Pass 2 |
| `$` | `$$` | Body text | Encoder Pass 3 (before dict) / Decoder Pass 2 (after dict) |
| `>` | `\>` | LINK/IMG display text only | Encoder Pass 4 / Decoder Pass 1 |
| `;` | `\;` | META values only | Encoder Pass 4 / Decoder Pass 1 |
| `\|` | `\\|` | TABLE cells, RETURNS blocks | Encoder Pass 4 / Decoder Pass 1 |
| `~` | `\~` | KV/PARAMS/DEF descriptions | Encoder Pass 4 / Decoder Pass 1 |
| `{` | `\{` | When literal `{` follows `@TAG` | Encoder Pass 4 / Decoder Pass 1 |
| `}` | `\}` | Inside block tags when literal `}` needed | Encoder Pass 4 / Decoder Pass 1 |
| `@` at line start | `\@` | Inside CODE blocks only | Encoder Pass 4 / Decoder Pass 1 |

---

## Implementation Phases

### Phase 1: Core Round-Trip (No Compression)

**Goal:** Convert Markdown ↔ CMDX structural tags with zero compression. Prove the round-trip invariant on basic markdown.

**Files to create:**

| File | Purpose |
|------|---------|
| `core/cmdx/AST.go` | All type definitions (Document, Node, NodeAttrs, TagType, domain types, EncoderOptions) |
| `core/cmdx/Escape.go` | All escape/unescape functions |
| `core/cmdx/Tags.go` | TagParser: parse `@TAG` syntax, dispatch, brace matching, code block handling |
| `core/cmdx/Encode.go` | Pass 1 (goldmark → AST) and Pass 4 (AST → CMDX serialization). Skip passes 2 and 3. |
| `core/cmdx/Decode.go` | Full pipeline: Parse → AST → serialize Markdown. Passes 2 and 3 are no-ops. |
| `core/cmdx/Normalize.go` | `DumpAST()` canonical tree serializer. `CompareASTs()` comparison function. |
| `core/cmdx/Escape_test.go` | Escape/unescape tests |
| `core/cmdx/Tags_test.go` | Tag parser tests |
| `core/cmdx/Normalize_test.go` | AST dump and comparison tests |
| `core/cmdx/Roundtrip_test.go` | Round-trip invariant tests |
| `core/cmdx/testdata/simple.md` | Basic fixture (copy from original plan) |
| `go.mod` | Add goldmark dependency |

**Implementation notes:**

- Start with the simplest possible input: a single heading and paragraph. Get the full encode→decode→compare cycle passing. Then add tags one at a time.
- The goldmark AST walker uses the `ast.Walk` function with a visitor callback. Each node type requires a case in a type switch.
- `DumpAST()` walks the goldmark AST and produces lines like `Heading[level=1]\n  Text "Hello"\n`. It strips `ast.BaseBlock.Lines` (source positions) and any alignment/title attributes.
- `CompareASTs()` calls `DumpAST()` on both inputs and returns `(equal bool, diff string, err error)`.

**Tags to implement in Phase 1:**

Structural: `@H1`–`@H6`, `@HR`, `@BR`, `@P`, `@BQ`
Inline: `@B{}`, `@I{}`, `@BI{}`, `@C{}`, `@S{}`, `@LINK{}`, `@IMG{}`
Block: `@CODE:lang`/`@/CODE`, `@UL{}`, `@OL{}`, `@TABLE{}`

**Tags NOT in Phase 1:** `@DICT{}`, `@META{}`, `@KV{}`, `@PARAMS{}`, `@ENDPOINT{}`, `@RETURNS{}`, `@DEF{}`, `@NOTE{}`, `@WARN{}`, `@TIP{}`, `@IMPORTANT{}`

**Phase 1 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestRoundTrip_simple` | Round-trip `simple.md` fixture |
| `TestRoundTrip_headingsOnly` | All 6 heading levels |
| `TestRoundTrip_paragraph` | Single paragraph, multi-paragraph, soft-wrapped paragraph |
| `TestRoundTrip_inlineFormatting` | Bold, italic, bold-italic, code, strikethrough |
| `TestRoundTrip_nestedInline` | `@B{some @I{italic} text}` nesting |
| `TestRoundTrip_links` | Links with various URLs, display text with `>` |
| `TestRoundTrip_images` | Images with alt text and URLs |
| `TestRoundTrip_codeBlock` | Fenced code with language, without language, indented code |
| `TestRoundTrip_codeBlockWithAtSign` | Code containing `@` at line start (escape verification) |
| `TestRoundTrip_codeBlockWithAtSlashCODE` | Code containing literal `@/CODE` text |
| `TestRoundTrip_unorderedList` | Flat list, nested list |
| `TestRoundTrip_orderedList` | Numbered list |
| `TestRoundTrip_mixedLists` | Ordered inside unordered and vice versa |
| `TestRoundTrip_blockquote` | Single-line, multi-line |
| `TestRoundTrip_table` | Basic table |
| `TestRoundTrip_horizontalRule` | `---` |
| `TestRoundTrip_empty` | Empty input |
| `TestRoundTrip_atSignInText` | Body text containing `@` (escape verification) |
| `TestRoundTrip_dollarSignInText` | Body text containing `$` (escape verification) |
| `TestEncode_headingLevels` | Verify `@H1`–`@H6` output for each heading level |
| `TestEncode_paragraphJoins` | Soft-wrapped lines joined into single `@P` |
| `TestEncode_codeBlockPreservesContent` | Whitespace, indentation preserved exactly |
| `TestDecode_invalidHeader` | Missing `@CMDX v1` returns error |
| `TestDecode_unknownTag` | Unknown `@FOO` tag returns error or is skipped gracefully |
| `TestDumpAST_stripsPositions` | Source positions not in dump output |
| `TestDumpAST_stripsAlignment` | Table alignment not in dump output |
| `TestCompareASTs_identical` | Same input → equal |
| `TestCompareASTs_different` | Different content → not equal, diff message |
| `TestCompareASTs_whitespaceNormalized` | Extra blank lines → still equal |

**Verification:**

```bash
go test -v -run TestRoundTrip ./core/cmdx/
go test -v -run TestEncode ./core/cmdx/
go test -v -run TestDecode ./core/cmdx/
go test -v -run TestDumpAST ./core/cmdx/
go test -v -run TestCompareASTs ./core/cmdx/
```

### Phase 2: Dictionary Compression

**Goal:** Add frequency analysis and dictionary-based string deduplication. Achieve measurable token savings on repetitive documents.

**Files to create/modify:**

| File | Action | Purpose |
|------|--------|---------|
| `core/cmdx/Dict.go` | Create | `BuildDictionary()`, candidate extraction, scoring, greedy selection |
| `core/cmdx/Dict_test.go` | Create | Dictionary builder unit tests |
| `core/cmdx/Encode.go` | Modify | Add Pass 3 (build dictionary, replace with `$N`) |
| `core/cmdx/Decode.go` | Modify | Add Pass 2 (expand `$N` references, unescape `$$`) |
| `core/cmdx/Roundtrip_test.go` | Modify | Add dictionary-enabled round-trip tests |

**Implementation notes:**

- The dictionary builder must use `sort.SliceStable` for determinism.
- Tie-breaking: when two candidates have the same score, the one whose first occurrence appears earlier in the document wins.
- The greedy selection loop: after selecting a candidate, scan all remaining candidates. For each, count how many of its occurrences overlap with the selected candidate's claimed positions. Reduce its frequency by the overlap count. Recalculate its score. If score drops to zero or negative, remove it.
- Dictionary indices are assigned after selection: sort selected entries by first-occurrence position, assign 0, 1, 2, ...
- The `$N` reference format: `$` followed by the decimal integer index. `$0`, `$1`, ..., `$49`. No leading zeros, no padding.

**Phase 2 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestBuildDictionary_noRepeats` | Input with no repeated strings → empty dictionary |
| `TestBuildDictionary_singleRepeat` | One string repeated 3 times → 1 entry |
| `TestBuildDictionary_multipleRepeats` | Multiple repeated strings → entries sorted by first occurrence |
| `TestBuildDictionary_minFrequency` | String appearing once is excluded |
| `TestBuildDictionary_minLength` | Short repeated string (< 10 chars) is excluded |
| `TestBuildDictionary_scoringPositive` | Entry with positive score is included |
| `TestBuildDictionary_scoringNegative` | Entry where overhead exceeds savings is excluded |
| `TestBuildDictionary_maxEntries` | Dictionary caps at MaxDictEntries |
| `TestBuildDictionary_overlapHandling` | Overlapping candidates: higher-scoring wins, lower is recalculated |
| `TestBuildDictionary_deterministic` | Same input twice → identical dictionary |
| `TestBuildDictionary_tieBreaking` | Equal scores → first-occurrence wins |
| `TestBuildDictionary_codeBlockExcluded` | Text inside code blocks is not considered for dictionary |
| `TestRoundTrip_withDictionary` | Round-trip with dictionary enabled |
| `TestRoundTrip_apiDocs` | Round-trip `api_docs.md` with dictionary |
| `TestEncode_dictOutput` | Verify `@DICT{...}` block format in output |
| `TestEncode_dictReferences` | Verify `$N` references appear in body where expected |
| `TestDecode_dictExpansion` | `$N` references expanded correctly |
| `TestDecode_dictExpansionInURL` | `$N` inside `@LINK{text>$0/path}` expanded |
| `TestDecode_dictExpansionInKVFields` | `$N` in KV item fields expanded (Phase 3 prep) |
| `TestDecode_dollarEscaping` | `$$` not treated as dictionary reference |
| `TestDecode_invalidDictRef` | `$99` when only 5 entries → error or literal pass-through |
| `TestStats_apiDocs` | Verify compression statistics against expected values |

**Verification:**

```bash
go test -v -run TestBuildDictionary ./core/cmdx/
go test -v -run TestRoundTrip ./core/cmdx/
go test -v -run TestStats ./core/cmdx/
```

### Phase 3: Domain-Specific Blocks

**Goal:** Detect and compress common documentation patterns (parameter tables, field docs, endpoint headers) into compact domain blocks.

**Files to modify:**

| File | Action | Purpose |
|------|--------|---------|
| `core/cmdx/Encode.go` | Modify | Add Pass 2 (domain pattern detection) |
| `core/cmdx/Decode.go` | Modify | Add Pass 3 (domain block → markdown table conversion) |
| `core/cmdx/Tags.go` | Modify | Add parsers for `@KV{}`, `@PARAMS{}`, `@ENDPOINT{}`, `@RETURNS{}`, `@DEF{}`, `@NOTE{}`, `@WARN{}`, `@TIP{}`, `@IMPORTANT{}` |
| `core/cmdx/Encode_test.go` | Modify | Domain detection tests |
| `core/cmdx/Decode_test.go` | Modify | Domain block decoding tests |
| `core/cmdx/Roundtrip_test.go` | Modify | Round-trip tests with domain blocks |

**Implementation notes:**

- Table pattern detection inspects `NodeAttrs.Headers` (case-insensitive match).
- KV detection: headers contain a name-like column AND `Type` AND `Description` AND NOT `Required`.
- PARAMS detection: headers contain a name-like column AND `Type` AND `Required` AND `Description`.
- Name-like columns: `Field`, `Key`, `Name`, `Parameter` (case-insensitive).
- Required mapping: `Yes`/`True`/`Y`/`1` → `R`. Everything else (`No`/`False`/`N`/`0`/empty) → `O`.
- ENDPOINT detection: H3 node whose text matches `^(GET|POST|PUT|DELETE|PATCH)\s+/`.
- Admonition detection: blockquote whose first child is a paragraph starting with `**Note:**`, `**Warning:**`, `**Tip:**`, `**Important:**` (case-insensitive).
- Admonition detection also covers GitHub-style `> [!NOTE]`, `> [!WARNING]`, `> [!TIP]`, `> [!IMPORTANT]` syntax.

**Phase 3 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestDetect_KVTable` | Table with Field/Type/Description → `@KV{}` |
| `TestDetect_KVTable_keyColumn` | Table with Key/Type/Description → `@KV{}` |
| `TestDetect_KVTable_nameColumn` | Table with Name/Type/Description → `@KV{}` |
| `TestDetect_KVTable_caseInsensitive` | Headers with mixed case → detected |
| `TestDetect_ParamsTable` | Table with Name/Type/Required/Description → `@PARAMS{}` |
| `TestDetect_ParamsTable_requiredMapping` | Yes→R, No→O, True→R, False→O |
| `TestDetect_ParamsTable_notKV` | PARAMS table not misidentified as KV (has Required column) |
| `TestDetect_Endpoint` | H3 `GET /users/{id}` → `@ENDPOINT{}` |
| `TestDetect_Endpoint_allMethods` | GET, POST, PUT, DELETE, PATCH all detected |
| `TestDetect_Endpoint_notH2` | H2 with endpoint text → NOT detected (H3 only) |
| `TestDetect_Endpoint_noMatch` | H3 without method → NOT detected |
| `TestDetect_StandardTable` | Table without matching pattern → `@TABLE{}` (unchanged) |
| `TestDetect_Admonition_boldNote` | `> **Note:** text` → `@WARN{}` or `@NOTE{}` |
| `TestDetect_Admonition_githubStyle` | `> [!NOTE]` → `@NOTE{}` |
| `TestDecode_KVToTable` | `@KV{}` → markdown table with Field/Type/Description headers |
| `TestDecode_ParamsToTable` | `@PARAMS{}` → markdown table with Required column |
| `TestDecode_EndpointToHeading` | `@ENDPOINT{GET /path}` → `### GET /path` |
| `TestDecode_ReturnsToText` | `@RETURNS{200:OK\|404:Not found}` → appropriate markdown |
| `TestDecode_DefToTable` | `@DEF{}` → markdown table with Term/Definition headers |
| `TestDecode_NoteToBlockquote` | `@NOTE{text}` → `> **Note:** text` |
| `TestRoundTrip_apiDocsFullPipeline` | Full round-trip of `api_docs.md` with all passes enabled |
| `TestRoundTrip_mixedContent` | Document with KV tables, plain tables, headings, paragraphs, code |
| `TestEncode_apiDocsMatchesExpected` | Encoder output for `api_docs.md` matches `api_docs.cmdx` |

**Verification:**

```bash
go test -v -run TestDetect ./core/cmdx/
go test -v -run TestDecode ./core/cmdx/
go test -v -run TestRoundTrip ./core/cmdx/
```

### Phase 4: CLI, Stats, Fuzz, Polish

**Goal:** Expose the library as CLI subcommands. Add fuzz testing. Validate with real codectx documentation.

**Files to create:**

| File | Purpose |
|------|---------|
| `core/cmdx/Stats.go` | `Analyze()` implementation |
| `core/cmdx/Stats_test.go` | Stats calculation tests |
| `core/cmdx/Fuzz_test.go` | Fuzz tests for round-trip safety |
| `cmds/cmdx/main.go` | Parent `codectx cmdx` command |
| `cmds/cmdx/encode.go` | `codectx cmdx encode [file] [-o output]` |
| `cmds/cmdx/decode.go` | `codectx cmdx decode [file] [-o output]` |
| `cmds/cmdx/stats.go` | `codectx cmdx stats [file]` |
| `cmds/cmdx/validate.go` | `codectx cmdx validate [file.cmdx]` |
| `cmds/cmdx/roundtrip.go` | `codectx cmdx roundtrip [file.md]` |
| `cmds/cmdx/main_test.go` | Command registration tests |
| `main.go` | Register `cmds/cmdx` command |

**CLI subcommands:**

```
codectx cmdx encode [--dict-max N] [--min-freq N] [--min-len N] [--no-domain] input.md [-o output.cmdx]
codectx cmdx decode input.cmdx [-o output.md]
codectx cmdx stats input.md              # Show compression statistics
codectx cmdx validate input.cmdx         # Validate CMDX format, report errors
codectx cmdx roundtrip input.md          # Encode, decode, compare — exit 0 if lossless
```

Default output is stdout. `-o` writes to file.

**Phase 4 test specifications:**

| Test Name | What It Verifies |
|-----------|-----------------|
| `TestCommand_metadata` | Command name, usage, subcommands registered |
| `TestEncode_subcommand_writesStdout` | Encode writes to stdout by default |
| `TestEncode_subcommand_writesFile` | `-o` flag writes to file |
| `TestDecode_subcommand_writesStdout` | Decode writes to stdout by default |
| `TestStats_subcommand_output` | Stats prints human-readable statistics |
| `TestValidate_validFile` | Valid CMDX → exit 0 |
| `TestValidate_invalidFile` | Invalid CMDX → exit 1 with error message |
| `TestRoundtrip_subcommand_success` | Lossless file → exit 0 |
| `TestAnalyze_apiDocs` | Known input → expected stats values |
| `TestAnalyze_emptyDocument` | Empty → 0 savings |
| `TestAnalyze_proseDocument` | Prose-heavy → lower savings than API docs |
| `FuzzRoundTrip` | Random markdown input → encode → decode → AST comparison passes |

**Fuzz test design:**

```go
func FuzzRoundTrip(f *testing.F) {
    // Seed corpus with known fixtures
    f.Add(readTestdata("simple.md"))
    f.Add(readTestdata("api_docs.md"))
    f.Add([]byte("# Hello\n\nWorld"))
    f.Add([]byte("| a | b |\n|---|---|\n| 1 | 2 |"))
    f.Add([]byte("```go\nfunc main() {}\n```"))
    f.Add([]byte("Text with @ and $ signs"))

    f.Fuzz(func(t *testing.T, input []byte) {
        encoded, err := cmdx.Encode(input)
        if err != nil {
            t.Skip() // Invalid markdown is fine to skip
        }
        decoded, err := cmdx.Decode(encoded)
        if err != nil {
            t.Fatalf("decode failed: %v", err)
        }
        equal, diff, err := cmdx.CompareASTs(input, decoded)
        if err != nil {
            t.Fatalf("compare failed: %v", err)
        }
        if !equal {
            t.Fatalf("round-trip mismatch:\n%s", diff)
        }
    })
}
```

**Real-world validation (manual, not automated):**

After Phase 4, run the encoder against actual codectx documentation to measure real-world savings:

```bash
# Test against codectx foundation docs
codectx cmdx stats docs/foundation/philosophy/README.md
codectx cmdx stats docs/foundation/documentation/README.md
codectx cmdx roundtrip docs/foundation/philosophy/README.md

# Test against codectx topic docs
codectx cmdx stats docs/topics/go/README.md

# Test against compiled output (if available)
codectx cmdx stats .codectx/objects/*.md
```

**Verification:**

```bash
go test -v ./core/cmdx/
go test -v ./cmds/cmdx/
go test -fuzz FuzzRoundTrip -fuzztime 60s ./core/cmdx/
just build
```

---

## Test Fixture Specifications

### `testdata/simple.md`

Copy from `docs/plans/lossless-markdown-compression/simple.md`. Covers: headings, paragraphs, bold, italic, code, links, images, unordered lists, ordered lists, blockquotes, fenced code blocks, horizontal rules, tables, strikethrough.

### `testdata/api_docs.md`

Copy from `docs/plans/lossless-markdown-compression/api_docs.md`. Contains: repeated URLs, repeated type names, parameter tables, field tables, endpoint headings, admonitions. High repetition profile ideal for dictionary + domain block compression.

### `testdata/api_docs.cmdx`

Copy from `docs/plans/lossless-markdown-compression/api_docs.cmdx`. Expected encoder output for `api_docs.md`. Used for exact output comparison tests (not just round-trip).

### `testdata/edge_cases.md`

Must be created. Should contain:

```markdown
# Edge Cases

## Special Characters

Text with @ sign and $ sign and > arrow.

Text with @@ double at and $$ double dollar.

## Code Block Edge Cases

Code with @ at line start:

` `` `go
@SomeAnnotation
func main() {}
@/CODE
fmt.Println("not a closing tag")
` `` `

## Nested Formatting

**Bold with *italic* inside** and *italic with **bold** inside*.

***Bold italic text***

A [link with > arrow](https://example.com/path?a>b) in text.

## Tables With No Pattern

| Animal | Sound |
|--------|-------|
| Cat    | Meow  |
| Dog    | Woof  |

## Empty Sections

##

---

## Dollar Signs

The cost is $5 per item. Variable $PATH is set. Use $$double for escaping.
```

### `testdata/prose.md`

Must be created. Prose-heavy content with minimal repetition. Used to validate that the encoder handles low-compression scenarios gracefully and to measure baseline savings on non-repetitive content.

### `testdata/empty.md`

Empty file (0 bytes). Tests empty input handling.

---

## Dependencies

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/yuin/goldmark` | latest | Markdown parsing (CommonMark + GFM). AST walker for encoder. |
| `github.com/yuin/goldmark/extension` | (included) | GFM extension: tables, strikethrough, task lists |
| `github.com/stretchr/testify` | (already in project) | Test assertions |
| `github.com/urfave/cli/v3` | (already in project) | CLI subcommands |

Only goldmark is a new dependency. It has zero transitive dependencies.

---

## Success Criteria

### Must-have (all phases)

1. `CompareASTs(input, Decode(Encode(input)))` returns `true` for all test fixtures.
2. `CompareASTs(input, Decode(Encode(input)))` returns `true` for all codectx foundation and topic docs.
3. Fuzz test runs for 60 seconds with no failures.
4. All tests pass: `go test ./core/cmdx/ ./cmds/cmdx/`.
5. Build succeeds: `just build`.
6. Encoder output for `api_docs.md` matches `api_docs.cmdx` exactly.

### Should-have (Phase 2+)

7. Token savings of 20%+ on `api_docs.md`.
8. Token savings of 10%+ on codectx foundation docs (prose-heavy).
9. Dictionary builder produces identical output on repeated runs (determinism).
10. Stats output shows meaningful compression breakdown (dictionary vs. domain vs. structural).

### Nice-to-have (Phase 4)

11. CLI subcommands have `--help` text matching usage patterns.
12. `codectx cmdx roundtrip` exits 0 for all codectx docs.
13. Fuzz test runs for 5 minutes with no failures.

---

## Risk Register

| Risk | Likelihood | Impact | Mitigation |
|------|-----------|--------|------------|
| goldmark AST walker misses a node type | Medium | High | Phase 1 round-trip tests catch this immediately. Add fixtures for every markdown feature. |
| Dictionary builder overlap logic has bugs | Medium | Medium | Extensive unit tests with known overlapping candidates. Determinism tests catch drift. |
| Code block escaping misses an edge case | Low | High | Fuzz testing. Explicit test for `@/CODE` inside code blocks. |
| Performance degrades on large documents | Low | Low | Defer optimization. Naive approach works for documents under 200KB. Cap sliding window at 100 chars. |
| goldmark version upgrade changes AST shape | Low | Medium | Pin goldmark version in go.mod. Test against specific version. |
| Nested inline formatting produces wrong markdown | Medium | Medium | Test `@B{some @I{italic} text}` explicitly. Verify markdown output matches expected nesting. |
| LINK/IMG with unusual URLs breaks parser | Low | Medium | Test URLs with `>`, `{`, `}`, query parameters, fragments. First-unescaped-`>` rule handles all cases. |

---

## Future Work (Not In Scope)

These are documented for context but are explicitly excluded from this plan.

1. **Compile pipeline integration** — `codectx compile --format cmdx` for compressed output. Requires design decisions about per-object vs. whole-output compression and cross-document dictionary sharing.
2. **Streaming API** — `EncodeReader`/`DecodeReader` for large documents. Not needed until documents exceed memory limits.
3. **Footnotes** — `@FN{id:text}` and `@FNREF{id}`. Not supported in v1.
4. **Math blocks** — `@MATH{...}` for LaTeX. Not supported in v1.
5. **Task lists** — Currently passed through with `[x]`/`[ ]` prefixes preserved in list item text. Could be formalized as `@TASK{x:text}`.
6. **External package** — Extracting `core/cmdx/` to a standalone `github.com/securacore/cmdx` repository for use outside codectx.
