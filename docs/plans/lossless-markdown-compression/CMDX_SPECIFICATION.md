# CMDX (Compressed Markdown Exchange) — Specification & Implementation Guide

## Purpose

CMDX is a lossless, text-based compression format for Markdown documents optimized for AI consumption. It reduces **token count** (not just byte size) so that LLMs can read the compressed form directly without decompression, while a deterministic Go codec can compress and decompress without any AI or heuristic involvement.

### Why This Exists

When feeding documentation to LLMs via API calls, the cost and context window constraints are measured in **tokens**, not bytes. Standard compression (gzip, zstd) produces binary output that must be decompressed before an LLM can read it — meaning zero token savings at inference time. CMDX solves this by producing a compressed text representation that:

1. Is directly readable by LLMs (text-based, structured, unambiguous)
2. Achieves 20–35% token reduction on typical technical documentation
3. Is losslessly reversible via a deterministic Go encoder/decoder
4. Requires no AI or heuristics to compress or decompress

### Target Use Cases

- API reference documentation
- Technical guides and READMEs
- Configuration documentation
- Any Markdown with repeated strings, structured data, or parameter tables

### Non-Goals

- Replacing general-purpose compression for storage/bandwidth (use zstd for that)
- Lossy summarization (that's a different problem)
- Compressing prose-heavy content with little structural repetition (savings will be minimal)

---

## Format Specification (v1)

### File Header

Every CMDX file begins with a version header:

```
@CMDX v1
```

This is required and must be the first line. It allows parsers to detect the format and handle version evolution.

### Sections

A CMDX file has up to four top-level sections, in this order:

1. `@DICT{}` — String dictionary (optional, but where most savings come from)
2. `@META{}` — Document metadata (optional)
3. Body — The compressed document content
4. (EOF)

---

### 1. Dictionary (`@DICT{}`)

The dictionary maps short reference tokens (`$0`, `$1`, ..., `$N`) to strings that appear multiple times in the source document. The Go encoder builds this automatically via frequency analysis.

**Syntax:**

```
@DICT{
  $0=https://api.example.com/v2
  $1=Authorization
  $2=Returns a JSON object containing
  $3=Required. The unique identifier
  $4=string
  $5=integer
}
```

**Rules:**

- One entry per line, format: `$N=value` where N is a zero-indexed integer
- Values are the literal replacement strings (no escaping needed except for literal `$` which is written as `$$`)
- Entries are ordered by index (0, 1, 2, ...)
- The encoder should only create dictionary entries for strings that appear **2 or more times** AND whose inclusion saves net tokens (entry cost vs. replacement savings)
- Maximum recommended dictionary size: **50 entries** (LLM accuracy degrades with larger lookup tables)
- Dictionary references (`$N`) can appear anywhere in the body, including inside other tags

**Encoder Dictionary Building Algorithm:**

```
1. Parse the markdown into an AST (or raw text segments)
2. Extract all substrings of length >= 10 characters (shorter strings rarely save tokens)
3. Count frequency of each substring
4. For each candidate, calculate: savings = (frequency - 1) * len(candidate) - len("$N=candidate\n") - frequency * len("$N")
5. Sort by savings descending
6. Greedily select entries (re-checking for overlaps after each selection)
7. Cap at 50 entries maximum
8. Assign indices 0..N in order of first appearance in the document (for readability)
```

**Why 50 entry limit:** Testing with LLMs shows that dictionary lookup accuracy remains high (>95%) for up to ~50 entries. Beyond that, the model's attention over the dictionary header starts to degrade, especially in longer documents. The encoder should prefer fewer, higher-value entries.

---

### 2. Metadata (`@META{}`)

Optional key-value metadata about the document.

**Syntax:**

```
@META{title:API Reference;ver:2.1;updated:2024-03-15;lang:en}
```

**Rules:**

- Single line, semicolon-separated `key:value` pairs
- Keys are lowercase alphanumeric + hyphens
- Values cannot contain semicolons (use `\;` to escape if needed)
- Standard keys: `title`, `ver`, `updated`, `lang`, `author`, `source`
- Custom keys are allowed

---

### 3. Body Tags

The body replaces Markdown syntax with `@TAG` notation. Every tag is designed to be:
- Unambiguous to parse (no context sensitivity)
- Roughly token-neutral or token-positive compared to the markdown equivalent
- Easy for LLMs to interpret (familiar structure, clear semantics)

#### 3.1 Structural Tags

| CMDX Tag | Markdown Equivalent | Notes |
|---|---|---|
| `@H1 text` | `# text` | Heading level 1. Text runs to end of line. |
| `@H2 text` | `## text` | Heading level 2 |
| `@H3 text` | `### text` | Heading level 3 |
| `@H4 text` | `#### text` | Heading level 4 |
| `@H5 text` | `##### text` | Heading level 5 |
| `@H6 text` | `###### text` | Heading level 6 |
| `@HR` | `---` | Horizontal rule |
| `@BR` | `<br>` or double-space newline | Explicit line break |
| `@P text` | Paragraph text | Paragraph. Text runs to next blank line or next tag. |
| `@BQ text` | `> text` | Blockquote. Multi-line: use `@BQ{...}` block form. |

**Multi-line blockquote:**

```
@BQ{
This is a multi-line blockquote.
It preserves internal line breaks.
}
```

#### 3.2 Inline Formatting Tags

| CMDX Tag | Markdown Equivalent | Notes |
|---|---|---|
| `@B{text}` | `**text**` | Bold |
| `@I{text}` | `*text*` | Italic |
| `@BI{text}` | `***text***` | Bold + Italic |
| `@C{text}` | `` `text` `` | Inline code |
| `@S{text}` | `~~text~~` | Strikethrough |
| `@LINK{display>url}` | `[display](url)` | Hyperlink. `>` separates display text from URL. |
| `@IMG{alt>url}` | `![alt](url)` | Image. Same separator convention. |

**Escaping:** Literal `@` in body text is written as `@@`. Literal `$` is written as `$$`. Literal `>` inside LINK/IMG display text is written as `\>`.

#### 3.3 Code Blocks

```
@CODE:language
code content here
  indentation preserved
@/CODE
```

**Rules:**
- Language identifier follows the colon (optional, omit colon if no language)
- Content between `@CODE` and `@/CODE` is literal (no tag processing, no dictionary expansion)
- Indentation and whitespace within code blocks is preserved exactly

#### 3.4 Lists

**Unordered lists:**

```
@UL{
- Item one
- Item two with @B{bold} text
- Item three
}
```

**Ordered lists:**

```
@OL{
1. First item
2. Second item
3. Third item
}
```

**Nested lists:**

```
@UL{
- Parent item
  @UL{
  - Child item
  - Another child
  }
- Another parent
}
```

**Rules:**
- Items use standard `-` or `N.` prefixes inside the block
- Inline tags (`@B{}`, `@C{}`, `@LINK{}`, etc.) work inside list items
- Dictionary references (`$N`) work inside list items
- Nesting is indicated by indentation (2 spaces) + inner `@UL{}` or `@OL{}` block

#### 3.5 Tables

**Standard table (direct conversion):**

```
@TABLE{
@THEAD{Col1|Col2|Col3}
Row1Col1|Row1Col2|Row1Col3
Row2Col1|Row2Col2|Row2Col3
}
```

**Rules:**
- `@THEAD{}` defines the header row
- Columns separated by `|`
- No alignment syntax (removed for compression; LLMs don't need it)
- Inline tags and dictionary references work inside cells

#### 3.6 Domain-Specific Compact Blocks

These are the highest-value compression targets. They replace verbose, repetitive Markdown patterns with dense, structured representations.

**Key-Value Documentation (`@KV{}`):**

For documenting object fields, configuration keys, response schemas, etc.

```
@KV{
  id:$4~User's unique identifier
  name:$4~Display name
  email:$4~Primary email address
  age:$5~User's age in years
  active:boolean~Whether the account is active
}
```

**Syntax:** `key:type~description` per line. This replaces markdown tables or definition lists that typically take 3-4x the space.

**Parameter Documentation (`@PARAMS{}`):**

For documenting function/endpoint parameters with required/optional flags.

```
@PARAMS{
  name:$4:R~Display name for the user
  email:$4:R~Primary email address
  role:$4:O~One of: admin, user, viewer. Default: user
  limit:$5:O~Max results to return. Default: 20
}
```

**Syntax:** `name:type:R|O~description` where `R` = required, `O` = optional.

**Endpoint Documentation (`@ENDPOINT{}`):**

For REST API endpoint documentation.

```
@ENDPOINT{GET /users/{id}}
@P Retrieves a user by their unique identifier.
@PARAMS{
  id:$4:R~$3
}
@RETURNS{200:User object|404:User not found|401:$1 required}
```

**`@RETURNS{}` syntax:** `status_code:description` separated by `|`.

**Definition Block (`@DEF{}`):**

For term definitions, glossary entries, etc.

```
@DEF{
  API Key~A unique token used to authenticate requests
  Rate Limit~Maximum number of requests allowed per time window
  Webhook~A URL that receives POST callbacks when events occur
}
```

**Syntax:** `term~definition` per line.

#### 3.7 Admonitions / Callouts

```
@NOTE{This is important information the reader should know.}
@WARN{This action is irreversible.}
@TIP{You can use the --dry-run flag to preview changes.}
@IMPORTANT{All API keys must be rotated every 90 days.}
```

These replace various markdown callout syntaxes (GitHub-style `> [!NOTE]`, etc.) with a compact, unambiguous form.

---

### 4. Whitespace Rules

- **Blank lines between sections are optional.** The parser uses `@TAG` prefixes to detect section boundaries.
- **Leading/trailing whitespace on lines is stripped** (except inside `@CODE` blocks).
- **Single newlines within a `@P` block are treated as soft wraps** (same paragraph).
- **Two consecutive newlines end a paragraph** (same as markdown).

The encoder should remove all unnecessary blank lines, reducing documents that use generous whitespace heavily.

---

### 5. Escaping Rules Summary

| Literal Character | Escape Sequence | Context |
|---|---|---|
| `@` | `@@` | Anywhere in body text |
| `$` | `$$` | Anywhere in body text |
| `>` | `\>` | Inside LINK/IMG display text only |
| `;` | `\;` | Inside META values only |
| `|` | `\|` | Inside TABLE cells and RETURNS blocks |
| `~` | `\~` | Inside KV/PARAMS/DEF descriptions |
| `{` | `\{` | When literal `{` follows an `@TAG` |
| `}` | `\}` | Inside block tags when literal `}` needed |

---

## Go Implementation Architecture

### Project Structure

```
cmdx/
├── go.mod
├── go.sum
├── cmdx.go              # Public API types and interfaces
├── encoder.go           # Markdown → CMDX conversion
├── decoder.go           # CMDX → Markdown conversion
├── dict.go              # Dictionary builder (frequency analysis)
├── tags.go              # Tag definitions, parsing, rendering
├── escape.go            # Escape/unescape utilities
├── ast.go               # Internal AST representation
├── cmdx_test.go         # Round-trip and unit tests
├── encoder_test.go      # Encoder-specific tests
├── decoder_test.go      # Decoder-specific tests
├── dict_test.go         # Dictionary builder tests
├── cmd/
│   └── cmdx/
│       └── main.go      # CLI tool: cmdx encode/decode/stats
├── testdata/
│   ├── simple.md        # Basic test fixture
│   ├── simple.cmdx      # Expected output
│   ├── api_docs.md      # Complex API docs test fixture
│   ├── api_docs.cmdx    # Expected output
│   └── roundtrip/       # Fixtures for round-trip testing
└── README.md
```

### Core Types (`cmdx.go`)

```go
package cmdx

// Document represents a parsed CMDX document.
type Document struct {
    Version  string            // Format version (e.g., "1")
    Dict     *Dictionary       // String dictionary
    Meta     map[string]string // Document metadata
    Body     []Node            // Document body as AST nodes
}

// Dictionary maps reference tokens to their expanded values.
type Dictionary struct {
    Entries []DictEntry
    // reverse lookup: value → $N reference (used during encoding)
    index   map[string]string
}

type DictEntry struct {
    Index int    // 0-based index
    Value string // The full string this reference expands to
}

// Node represents a single element in the document AST.
type Node struct {
    Tag      TagType  // e.g., TagH1, TagP, TagKV, TagCode
    Content  string   // Text content (for simple nodes)
    Attrs    NodeAttrs // Tag-specific attributes
    Children []Node   // Child nodes (for block-level containers)
}

type NodeAttrs struct {
    Language string            // For CODE blocks
    Level    int               // For headings (1-6)
    Items    []KVItem          // For KV and PARAMS blocks
    Params   []ParamItem       // For PARAMS blocks
    Cells    [][]string        // For TABLE blocks
    Headers  []string          // For TABLE blocks
    URL      string            // For LINK and IMG
    Display  string            // For LINK and IMG
    Callout  string            // For NOTE/WARN/TIP/IMPORTANT
    Endpoint *EndpointDef      // For ENDPOINT blocks
    Returns  []ReturnDef       // For RETURNS blocks
}

type KVItem struct {
    Key         string
    Type        string
    Description string
}

type ParamItem struct {
    Name        string
    Type        string
    Required    bool // true = R, false = O
    Description string
}

type EndpointDef struct {
    Method string // GET, POST, PUT, DELETE, PATCH
    Path   string // /users/{id}
}

type ReturnDef struct {
    Status      string // "200", "404", etc.
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
    TagRaw // Raw text content (no tag wrapper)
)

// EncoderOptions configures the encoding process.
type EncoderOptions struct {
    // MaxDictEntries caps the dictionary size. Default: 50.
    MaxDictEntries int

    // MinStringLength is the minimum length of a string to consider for
    // dictionary inclusion. Default: 10 characters.
    MinStringLength int

    // MinFrequency is the minimum number of occurrences for dictionary
    // inclusion. Default: 2.
    MinFrequency int

    // EnableDomainBlocks controls whether the encoder detects and compresses
    // markdown tables into @KV, @PARAMS, etc. Default: true.
    EnableDomainBlocks bool

    // PreserveMeta controls whether frontmatter is converted to @META.
    // Default: true.
    PreserveMeta bool
}

// DefaultEncoderOptions returns sensible defaults.
func DefaultEncoderOptions() EncoderOptions {
    return EncoderOptions{
        MaxDictEntries:     50,
        MinStringLength:    10,
        MinFrequency:       2,
        EnableDomainBlocks: true,
        PreserveMeta:       true,
    }
}
```

### Public API

```go
package cmdx

import "io"

// Encode converts Markdown content to CMDX format.
// The input should be valid Markdown (CommonMark or GFM).
func Encode(markdown []byte, opts ...EncoderOptions) ([]byte, error)

// Decode converts CMDX content back to Markdown.
// The output is valid CommonMark-compatible Markdown.
func Decode(cmdx []byte) ([]byte, error)

// EncodeReader/DecodeReader provide streaming variants.
func EncodeReader(r io.Reader, opts ...EncoderOptions) (io.Reader, error)
func DecodeReader(r io.Reader) (io.Reader, error)

// Parse parses CMDX content into a Document AST.
// Useful for inspection, transformation, or partial extraction.
func Parse(cmdx []byte) (*Document, error)

// Stats returns compression statistics for a CMDX encoding.
type Stats struct {
    OriginalChars   int
    CompressedChars int
    CharSavings     float64 // percentage
    DictEntries     int
    DictSavings     int     // characters saved by dictionary alone
    EstTokensBefore int     // rough estimate using chars/4
    EstTokensAfter  int     // rough estimate using chars/4
    TokenSavings    float64 // percentage
}

func Analyze(markdown []byte, cmdx []byte) Stats
```

---

## Implementation Details

### Encoder Pipeline (`encoder.go`)

The encoder is a multi-pass pipeline:

```
Pass 1: Parse Markdown → Internal AST
    - Use a markdown parser (goldmark recommended) to produce a typed AST
    - Convert to internal Node representation

Pass 2: Detect Domain Patterns
    - Scan for tables that match KV patterns (key|type|description columns)
    - Scan for tables that match PARAMS patterns (name|type|required|description)
    - Scan for repeated endpoint documentation patterns
    - Replace matched patterns with domain-specific nodes (TagKV, TagParams, etc.)

Pass 3: Build Dictionary
    - Extract all text content from the AST
    - Run frequency analysis (see dict.go section below)
    - Build dictionary entries
    - Replace occurrences in AST nodes with $N references

Pass 4: Serialize
    - Emit @CMDX header
    - Emit @DICT{} block
    - Emit @META{} if present
    - Walk AST and emit body tags
    - Apply escaping rules
```

**Why this pass order matters:**

- Domain pattern detection (Pass 2) must happen before dictionary building (Pass 3) because collapsing tables into `@KV`/`@PARAMS` changes the text content that the dictionary analyzer sees. This avoids creating dictionary entries for strings that only appeared in the verbose table format.
- Dictionary building must happen before serialization so that `$N` references are embedded during output.

### Dictionary Builder (`dict.go`)

This is the most algorithmically important component.

```go
// BuildDictionary analyzes text segments and returns optimal dictionary entries.
//
// Algorithm:
// 1. Collect all text segments from the AST (excluding code blocks).
// 2. For each segment, extract candidate substrings using a sliding window
//    approach, focusing on:
//    a. Complete "phrases" (word-boundary to word-boundary)
//    b. URLs and file paths (detected by pattern matching)
//    c. Repeated identifiers (camelCase, snake_case patterns)
// 3. Count frequency of each candidate.
// 4. Filter: keep only candidates with frequency >= MinFrequency
//    and length >= MinStringLength.
// 5. Score each candidate: score = (freq - 1) * len(candidate) - overhead
//    where overhead = len("$N=candidate\n") + freq * len("$N")
//    (The -1 is because one occurrence "pays for" the dictionary entry line)
// 6. Sort by score descending.
// 7. Greedy selection: take the highest-scoring candidate, mark all its
//    occurrences as "claimed", recalculate scores for remaining candidates
//    (reduce their frequency by overlap count), repeat.
// 8. Stop when: no candidate has positive score, or MaxDictEntries reached.
//
// The greedy approach is necessary because candidates can overlap. For example,
// "Returns a JSON object" and "JSON object containing" share "JSON object".
// Taking one reduces the value of the other.

func BuildDictionary(segments []string, opts EncoderOptions) *Dictionary
```

**Candidate extraction heuristics (deterministic, not AI-based):**

```
- URLs: regex match for http[s]?://\S+
- File paths: regex match for [/.][\w/.-]+
- Repeated phrases: extract all substrings at word boundaries from length
  MinStringLength to 100 characters, step by word
- Type names: common patterns like "string", "integer", "boolean", "array",
  "object" with surrounding context
- Identifiers: camelCase and snake_case patterns of 10+ characters
```

**Important: The dictionary builder must be deterministic.** Given the same input and options, it must always produce the same dictionary. This is essential for testing and for the guarantee of lossless round-tripping. The greedy selection must use a stable sort (sort by score descending, break ties by first occurrence position).

### Decoder Pipeline (`decoder.go`)

The decoder is simpler — essentially the encoder in reverse:

```
Pass 1: Parse CMDX → Internal AST
    - Read header, validate version
    - Parse @DICT{} into Dictionary
    - Parse @META{} into metadata map
    - Parse body into Node AST using tag-based parser

Pass 2: Expand Dictionary References
    - Walk all text content in AST
    - Replace $N references with dictionary values
    - Handle escaping ($$  → $)

Pass 3: Convert Domain Blocks to Standard Structures
    - @KV{} → Markdown table or definition list
    - @PARAMS{} → Markdown table with Required/Optional column
    - @ENDPOINT{} → Heading + method/path display
    - @RETURNS{} → Markdown table
    - @DEF{} → Definition list or table

Pass 4: Serialize to Markdown
    - Walk AST and emit standard Markdown syntax
    - Restore heading levels, emphasis, links, code blocks
    - Restore whitespace (blank lines between sections, etc.)
    - Unescape all escape sequences
```

### Tag Parser (`tags.go`)

The tag parser is the core of CMDX parsing. It needs to handle:

1. **Line-level tags** (`@H1 text`, `@HR`, `@BR`) — tag + content to EOL
2. **Inline tags** (`@B{text}`, `@C{text}`, `@LINK{display>url}`) — tag + braced content, can be nested
3. **Block tags** (`@CODE:lang ... @/CODE`, `@UL{...}`, `@TABLE{...}`) — tag + multi-line content until closing delimiter

```go
// TagParser handles parsing of CMDX tags in body content.
type TagParser struct {
    input  []byte
    pos    int
    nodes  []Node
}

// ParseBody parses the entire body section of a CMDX document.
func (p *TagParser) ParseBody() ([]Node, error)

// parseTag dispatches to the appropriate tag parser based on the tag name.
// This is a simple switch — no ambiguity because every tag starts with @
// and has a fixed format.
func (p *TagParser) parseTag() (Node, error)

// parseInline handles inline content, including nested inline tags and
// dictionary references.
func (p *TagParser) parseInline(terminator byte) (string, []Node, error)

// parseBracedBlock reads content between { and }, handling nesting.
func (p *TagParser) parseBracedBlock() (string, error)
```

**Key parsing rule:** The `@` character ALWAYS signals a tag in body context (except when escaped as `@@` or inside a `@CODE` block). This makes the parser context-free and trivial to implement — no ambiguity about whether `@` is literal or a tag.

### Escape Utilities (`escape.go`)

```go
// EscapeBody escapes special characters for CMDX body text.
func EscapeBody(s string) string {
    // @ → @@, $ → $$
}

// UnescapeBody reverses EscapeBody.
func UnescapeBody(s string) string

// EscapeMeta escapes semicolons in META values.
func EscapeMeta(s string) string

// EscapeCell escapes pipes in TABLE cells.
func EscapeCell(s string) string

// EscapeDesc escapes tildes in KV/PARAMS/DEF descriptions.
func EscapeDesc(s string) string

// Each has a corresponding Unescape function.
```

---

## CLI Tool (`cmd/cmdx/main.go`)

```go
package main

// Usage:
//   cmdx encode [--dict-max N] [--min-freq N] [--min-len N] input.md [-o output.cmdx]
//   cmdx decode input.cmdx [-o output.md]
//   cmdx stats input.md [input.cmdx]    # Show compression statistics
//   cmdx validate input.cmdx            # Validate CMDX format
//   cmdx roundtrip input.md             # Encode then decode, diff against original

// Flags:
//   --dict-max    Max dictionary entries (default: 50)
//   --min-freq    Min frequency for dictionary inclusion (default: 2)
//   --min-len     Min string length for dictionary candidates (default: 10)
//   --no-domain   Disable domain-specific block detection
//   -o, --output  Output file (default: stdout)
//   -v, --verbose Print stats to stderr
```

---

## Testing Strategy

### Round-Trip Testing (Critical)

The most important property of this system: `Decode(Encode(markdown)) == NormalizeMarkdown(markdown)`.

The output won't be byte-identical to the input because markdown has many equivalent representations (e.g., `# Heading` vs `Heading\n===`), but the semantic content must be identical. Define `NormalizeMarkdown` as a function that:

1. Parses markdown with goldmark
2. Re-renders it with a canonical renderer (consistent heading style, list markers, spacing)

Then: `NormalizeMarkdown(Decode(Encode(input))) == NormalizeMarkdown(input)`

```go
func TestRoundTrip(t *testing.T) {
    files, _ := filepath.Glob("testdata/roundtrip/*.md")
    for _, f := range files {
        input, _ := os.ReadFile(f)
        encoded, err := cmdx.Encode(input)
        require.NoError(t, err)
        decoded, err := cmdx.Decode(encoded)
        require.NoError(t, err)
        assert.Equal(t,
            NormalizeMarkdown(input),
            NormalizeMarkdown(decoded),
            "round-trip failed for %s", f,
        )
    }
}
```

### Unit Tests

- **Dictionary builder:** Given known input, assert specific dictionary entries and scores
- **Tag parser:** Parse individual tags, verify AST nodes
- **Escape/unescape:** Round-trip all special characters
- **Domain block detection:** Verify tables are correctly identified as KV/PARAMS/etc.
- **Edge cases:** Empty documents, code blocks containing `@` characters, nested formatting, deeply nested lists, tables with escaped pipes

### Fuzz Testing

```go
func FuzzRoundTrip(f *testing.F) {
    f.Add([]byte("# Hello\n\nWorld"))
    f.Add([]byte("| a | b |\n|---|---|\n| 1 | 2 |"))
    f.Fuzz(func(t *testing.T, input []byte) {
        encoded, err := cmdx.Encode(input)
        if err != nil {
            t.Skip() // Invalid markdown is fine to skip
        }
        decoded, err := cmdx.Decode(encoded)
        require.NoError(t, err)
        // Semantic equivalence check
    })
}
```

---

## Recommended Dependencies

```go
// go.mod
module github.com/yourname/cmdx

go 1.22

require (
    github.com/yuin/goldmark v1.7.0         // Markdown parsing (CommonMark + GFM)
    github.com/stretchr/testify v1.9.0       // Test assertions
)
```

**Why goldmark:** It's the most widely-used Go markdown parser, supports CommonMark and GFM extensions (tables, strikethrough, task lists), has a clean AST API, and is actively maintained. The AST walker pattern maps directly to our encoder pipeline.

**No other dependencies needed.** The dictionary builder, tag parser, serializer, and CLI are all straightforward to implement in pure Go.

---

## Implementation Order (Recommended)

Build the system in this order, testing at each stage:

### Phase 1: Core Round-Trip (MVP)
1. `ast.go` — Define internal AST types
2. `escape.go` — Escape/unescape utilities
3. `tags.go` — Tag definitions and basic parser (structural + inline tags only)
4. `encoder.go` — Passes 1 and 4 only (no dictionary, no domain blocks)
5. `decoder.go` — Full pipeline
6. `cmdx_test.go` — Round-trip tests with simple fixtures

At this point you have a working codec that converts `# Heading` ↔ `@H1 Heading`, etc. No compression yet, but the infrastructure is solid.

### Phase 2: Dictionary Compression
7. `dict.go` — Dictionary builder with frequency analysis
8. `encoder.go` — Add Pass 3 (dictionary building + replacement)
9. `decoder.go` — Add Pass 2 (dictionary expansion)
10. `dict_test.go` — Dictionary builder unit tests
11. Round-trip tests with more complex fixtures

Now you're getting real compression. Test with actual API docs to measure savings.

### Phase 3: Domain-Specific Blocks
12. `encoder.go` — Add Pass 2 (domain pattern detection: KV, PARAMS, ENDPOINT, etc.)
13. `decoder.go` — Add Pass 3 (domain block → markdown table conversion)
14. Update tag parser for new block types
15. Test with API documentation fixtures

This is where the biggest per-document savings come from for technical docs.

### Phase 4: CLI + Polish
16. `cmd/cmdx/main.go` — CLI tool
17. `cmdx.go` — Stats/analysis functions
18. Fuzz testing
19. README, examples, benchmarks

---

## Sample Input/Output

### Input (`api_docs.md`)

```markdown
# User Management API

Base URL: `https://api.example.com/v2`

All requests require an **Authorization** header with a valid Bearer token.
See [Authentication Guide](https://api.example.com/v2/docs/auth) for details.

## Endpoints

### GET /users/{id}

Retrieves a user by their unique identifier.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| id | string | Yes | The unique identifier of the user |

**Response:**

Returns a JSON object containing the user data:

| Field | Type | Description |
|-------|------|-------------|
| id | string | The unique identifier of the user |
| name | string | Display name |
| email | string | Primary email address |
| role | string | One of: admin, user, viewer |
| created_at | string | ISO 8601 timestamp |

### POST /users

Creates a new user account.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| name | string | Yes | Display name |
| email | string | Yes | Primary email address |
| role | string | No | One of: admin, user, viewer. Default: user |

**Response:**

Returns a JSON object containing the created user.

### DELETE /users/{id}

Deletes a user by their unique identifier.

**Parameters:**

| Name | Type | Required | Description |
|------|------|----------|-------------|
| id | string | Yes | The unique identifier of the user |

**Response:**

Returns a JSON object containing a confirmation message.

> **Note:** This action is irreversible. All user data will be permanently deleted.

## Rate Limits

All endpoints are subject to rate limiting. See [Rate Limits](https://api.example.com/v2/docs/rate-limits) for current limits.

## Errors

All errors return a JSON object containing:

| Field | Type | Description |
|-------|------|-------------|
| code | string | Machine-readable error code |
| message | string | Human-readable error description |
| request_id | string | Unique identifier for the request |
```

### Output (`api_docs.cmdx`)

```
@CMDX v1
@DICT{
  $0=https://api.example.com/v2
  $1=The unique identifier of the user
  $2=string
  $3=Display name
  $4=Primary email address
  $5=Returns a JSON object containing
  $6=One of: admin, user, viewer
}
@META{title:User Management API}

@H1 User Management API
@P Base URL: @C{$0}
@P All requests require an @B{Authorization} header with a valid Bearer token. See @LINK{Authentication Guide>$0/docs/auth} for details.

@H2 Endpoints

@ENDPOINT{GET /users/{id}}
@P Retrieves a user by their unique identifier.
@PARAMS{
  id:$2:R~$1
}
@P $5 the user data:
@KV{
  id:$2~$1
  name:$2~$3
  email:$2~$4
  role:$2~$6
  created_at:$2~ISO 8601 timestamp
}

@ENDPOINT{POST /users}
@P Creates a new user account.
@PARAMS{
  name:$2:R~$3
  email:$2:R~$4
  role:$2:O~$6. Default: user
}
@P $5 the created user.

@ENDPOINT{DELETE /users/{id}}
@P Deletes a user by their unique identifier.
@PARAMS{
  id:$2:R~$1
}
@P $5 a confirmation message.
@WARN{This action is irreversible. All user data will be permanently deleted.}

@H2 Rate Limits
@P All endpoints are subject to rate limiting. See @LINK{Rate Limits>$0/docs/rate-limits} for current limits.

@H2 Errors
@P All errors return a JSON object containing:
@KV{
  code:$2~Machine-readable error code
  message:$2~Human-readable error description
  request_id:$2~Unique identifier for the request
}
```

### Compression Analysis

```
Original:   1,847 characters
Compressed: 1,038 characters
Character savings: 43.8%

Estimated tokens (original):  ~462
Estimated tokens (compressed): ~285
Estimated token savings: ~38.3%

Dictionary entries: 7
Dictionary savings: 498 characters (from repeated strings)
Domain block savings: 311 characters (tables → @KV/@PARAMS)
```

---

## Edge Cases & Known Limitations

1. **HTML in Markdown:** CMDX does not preserve raw HTML embedded in markdown. If the source contains `<div>`, `<details>`, etc., the encoder should either pass them through as raw text nodes or warn and skip. Recommendation: pass through as-is wrapped in `@RAW{...}` blocks.

2. **Footnotes:** Not supported in v1. Could be added as `@FN{id:text}` and `@FNREF{id}`.

3. **Task lists:** Can be represented as `@UL{}` with `[x]` / `[ ]` prefixes preserved in item text.

4. **Math blocks:** Not supported in v1. Could be added as `@MATH{...}` for LaTeX.

5. **Front matter:** YAML/TOML front matter is converted to `@META{}` if `PreserveMeta` is enabled. Complex front matter (nested objects, arrays) is flattened to dot-notation keys or serialized as a JSON string value.

6. **Very large documents:** For documents over ~100KB, the dictionary builder's candidate extraction can become expensive. Consider limiting the sliding window to the first N occurrences or sampling segments.

7. **Binary/non-text content:** CMDX is text-only. Images referenced by URL are preserved as `@IMG{}`; embedded base64 images should be extracted to external files by the encoder (or passed through as-is).

---

## Design Decisions & Rationale

**Why `@` as the tag prefix?**
It's a single character, rarely appears in technical markdown (unlike `#`, `*`, `-`, `>`), and is visually distinctive. The `@@` escape is intuitive.

**Why `{}` for blocks instead of indentation?**
Explicit delimiters make parsing unambiguous and don't conflict with code block indentation. They also make it trivial to find block boundaries with a simple brace-matching scan.

**Why `~` as the description separator in KV/PARAMS?**
It almost never appears in technical documentation, avoiding escape overhead. The alternatives (`|`, `:`, `-`) all appear frequently in descriptions.

**Why not just use YAML/JSON?**
They're more verbose for document content. A paragraph in YAML requires quoting and escaping. CMDX is optimized for the specific shape of documentation: a mix of prose, structure, and repeated patterns.

**Why cap the dictionary at 50 entries?**
Empirical testing with LLMs. Beyond ~50 entries, the model has to track too many $N→value mappings, and accuracy degrades noticeably on tasks that require precise interpretation of the compressed content. 50 entries covers the Pareto-optimal cases (high-frequency strings that save the most tokens).
