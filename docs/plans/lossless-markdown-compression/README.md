# CMDX — Compressed Markdown Exchange

A lossless, text-based compression format for Markdown that LLMs can read directly in compressed form, reducing token consumption by 20–35% on typical technical documentation.

---

## Quick Orientation

This repository contains a **specification and project scaffold** — not a finished implementation. The Go source files need to be written. Everything you need to build them is documented here. This README tells you how to navigate the project, what to read, and in what order.

### File Map

```
cmdx/
├── README.md                        ← You are here. Start with this.
├── CLAUDE.md                        ← Project context file for Claude Code.
│                                      Read this second — it's a concise summary
│                                      of the architecture and build phases.
├── CMDX_SPECIFICATION.md            ← The complete spec. Read this third.
│                                      This is the primary reference document
│                                      (~1,100 lines) containing everything:
│                                      format rules, Go types, algorithms,
│                                      pipeline design, sample I/O, and rationale.
├── go.mod                           ← Go module definition. Lists dependencies
│                                      (goldmark for markdown parsing, testify
│                                      for test assertions).
└── testdata/
    ├── api_docs.md                  ← Sample input: a realistic API reference
    │                                  document in standard Markdown.
    ├── api_docs.cmdx                ← Expected output: the same document
    │                                  compressed into CMDX format. Compare
    │                                  these two files side-by-side to understand
    │                                  what the encoder should produce.
    └── roundtrip/
        └── simple.md                ← A minimal Markdown file covering basic
                                       syntax (headings, lists, tables, code
                                       blocks, inline formatting). Use this
                                       for early round-trip testing.
```

---

## Reading Order

Follow this sequence. Each step builds on the previous one.

### Step 1: Read this README (5 minutes)

You're doing this now. It gives you the big picture — what the project is, how it's organized, and what you'll be building.

### Step 2: Read `CLAUDE.md` (2 minutes)

This is the short-form project guide. It lists the critical invariant (the round-trip property that must always hold), the four implementation phases, the high-level encode/decode architecture, and the dependencies. If you're using Claude Code, this file is automatically picked up as project context.

### Step 3: Compare `testdata/api_docs.md` and `testdata/api_docs.cmdx` (10 minutes)

Before reading the spec, open these two files side by side. Walk through them line by line and observe:

- How Markdown headings (`#`, `##`, `###`) become `@H1`, `@H2`, `@H3`
- How the `@DICT{}` block at the top captures repeated strings (`string`, `https://api.example.com/v2`, `The unique identifier of the user`, etc.) and replaces them with `$0`, `$1`, `$2`...
- How Markdown tables with parameter documentation collapse into compact `@PARAMS{}` blocks
- How Markdown tables with field documentation collapse into `@KV{}` blocks
- How links become `@LINK{display>url}`
- How the blockquote callout becomes `@WARN{}`

This concrete example will make the spec much easier to absorb.

### Step 4: Read `CMDX_SPECIFICATION.md` (30–45 minutes)

This is the main document. Read it end to end. It's organized in this order:

1. **Purpose & Non-Goals** — What this format is for and what it isn't
2. **Format Specification** — Every tag, syntax rule, and escaping convention
3. **Go Implementation Architecture** — Project structure, type definitions, public API
4. **Encoder Pipeline** — The four-pass encoding process with rationale for pass ordering
5. **Dictionary Builder** — The frequency analysis algorithm (the most important compression component)
6. **Decoder Pipeline** — The four-pass decoding process
7. **Tag Parser** — How the `@TAG` syntax is parsed (context-free, no ambiguity)
8. **CLI Tool** — Command-line interface design
9. **Testing Strategy** — Round-trip tests, unit tests, fuzz tests
10. **Sample Input/Output** — Full worked example with compression statistics
11. **Edge Cases & Limitations** — What v1 doesn't handle
12. **Design Decisions & Rationale** — Why `@`, why `{}`, why `~`, why cap at 50 dictionary entries

Pay special attention to:

- The **Go type definitions** in the "Core Types" section — these are the data structures you'll be implementing against
- The **Encoder Pipeline** section — the four-pass design and why the passes must execute in that specific order
- The **Dictionary Builder Algorithm** — this is the most algorithmically complex component and where most compression savings come from
- The **Implementation Order** section — it prescribes a phased approach so you always have a working, testable system

### Step 5: Review `testdata/roundtrip/simple.md` (2 minutes)

This is a minimal fixture for early testing. It covers headings, paragraphs, bold/italic/code, links, images, unordered and ordered lists, blockquotes, code blocks, horizontal rules, tables, and strikethrough. Your Phase 1 encoder should handle all of these.

---

## Implementation Process

### Prerequisites

- Go 1.22 or later
- A working `$GOPATH` or module-aware environment

### Initial Setup

```bash
cd cmdx/
go mod tidy          # Downloads goldmark and testify
```

If `go mod tidy` complains about missing sum entries, that's expected — the `go.sum` file hasn't been generated yet. Running `tidy` will create it.

### Phase 1: Core Round-Trip (Build First)

**Goal:** Convert Markdown ↔ CMDX structural tags with no compression. Prove the round-trip invariant.

**Files to create:**

| File           | Purpose                    | Key Concerns                                                                                                                                                                         |
| -------------- | -------------------------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `ast.go`       | Internal AST node types    | Should represent both Markdown and CMDX semantics. The `Node`, `NodeAttrs`, and `TagType` types from the spec go here.                                                               |
| `escape.go`    | Escape/unescape functions  | Handle `@`→`@@`, `$`→`$$`, and context-specific escapes (`~`, `\|`, etc.). Write these early because everything else depends on them.                                                |
| `tags.go`      | Tag definitions and parser | The `TagParser` that reads `@TAG` syntax. Start with line-level tags (`@H1`, `@HR`) and inline tags (`@B{}`, `@C{}`, `@LINK{}`), then add block tags (`@CODE`, `@UL{}`, `@TABLE{}`). |
| `encoder.go`   | Markdown → CMDX            | In Phase 1, implement only Pass 1 (Markdown → AST via goldmark) and Pass 4 (AST → CMDX serialization). Skip dictionary and domain detection for now.                                 |
| `decoder.go`   | CMDX → Markdown            | Full pipeline: parse tags → AST → serialize Markdown. In Phase 1, dictionary expansion and domain block conversion are no-ops.                                                       |
| `cmdx_test.go` | Round-trip tests           | The critical test: `NormalizeMarkdown(Decode(Encode(input))) == NormalizeMarkdown(input)`. Start with `testdata/roundtrip/simple.md`.                                                |

**How to verify:**

```bash
go test -v -run TestRoundTrip ./...
```

The test should encode `simple.md` to CMDX, decode it back, and confirm semantic equivalence.

**Recommended approach:** Start with the simplest possible Markdown input (a single heading and paragraph), get the full encode→decode→compare cycle working, then incrementally add support for more Markdown syntax. Don't try to handle all tag types at once.

### Phase 2: Dictionary Compression (Build Second)

**Goal:** Add the dictionary system for meaningful token savings.

**Files to create/modify:**

| File           | Purpose                                                                      |
| -------------- | ---------------------------------------------------------------------------- |
| `dict.go`      | Dictionary builder — frequency analysis, candidate scoring, greedy selection |
| `encoder.go`   | Add Pass 3: build dictionary, replace repeated strings with `$N` references  |
| `decoder.go`   | Add Pass 2: expand `$N` references back to full strings                      |
| `dict_test.go` | Unit tests for dictionary building                                           |

**How to verify:**

```bash
# Unit tests for dictionary builder
go test -v -run TestDictionary ./...

# Round-trip tests should still pass with dictionary enabled
go test -v -run TestRoundTrip ./...
```

**Key implementation detail:** The dictionary builder must be deterministic. Same input, same options → same dictionary every time. Use stable sorting and break ties by first-occurrence position in the source document.

Test with `testdata/api_docs.md` — it has high repetition and should show significant dictionary savings. Compare your output against `testdata/api_docs.cmdx`.

### Phase 3: Domain-Specific Blocks (Build Third)

**Goal:** Detect and compress common documentation patterns (parameter tables, field docs, endpoint headers) into compact blocks.

**Files to modify:**

| File         | Change                                                                                                  |
| ------------ | ------------------------------------------------------------------------------------------------------- |
| `encoder.go` | Add Pass 2: scan for tables matching KV/PARAMS/ENDPOINT patterns, replace with domain nodes             |
| `decoder.go` | Add Pass 3: convert `@KV{}`, `@PARAMS{}`, `@ENDPOINT{}`, `@RETURNS{}`, `@DEF{}` back to Markdown tables |
| `tags.go`    | Add parsers for new block types                                                                         |

**Pattern detection rules (deterministic, not heuristic):**

- A table with columns named `Field/Key/Name`, `Type`, `Description` → `@KV{}`
- A table with columns named `Name/Parameter`, `Type`, `Required`, `Description` → `@PARAMS{}`
- An H3 heading matching `METHOD /path` (where METHOD is GET/POST/PUT/DELETE/PATCH) → `@ENDPOINT{}`

**How to verify:**

```bash
# Full test suite including domain block conversion
go test -v ./...

# Compare against expected output
diff <(go run ./cmd/cmdx encode testdata/api_docs.md) testdata/api_docs.cmdx
```

### Phase 4: CLI and Polish (Build Last)

**Goal:** A usable command-line tool plus production hardening.

**Files to create:**

| File               | Purpose                                                                            |
| ------------------ | ---------------------------------------------------------------------------------- |
| `cmd/cmdx/main.go` | CLI: `cmdx encode`, `cmdx decode`, `cmdx stats`, `cmdx validate`, `cmdx roundtrip` |
| `cmdx.go`          | Public API: `Encode()`, `Decode()`, `Parse()`, `Analyze()` as clean entry points   |
| `encoder_test.go`  | Encoder-specific edge case tests                                                   |
| `decoder_test.go`  | Decoder-specific edge case tests                                                   |

**Add fuzz testing:**

```bash
go test -fuzz FuzzRoundTrip -fuzztime 60s ./...
```

**How to use the finished CLI:**

```bash
# Compress a markdown file
cmdx encode docs/api.md -o docs/api.cmdx

# Decompress back to markdown
cmdx decode docs/api.cmdx -o docs/api.md

# See compression statistics
cmdx stats docs/api.md

# Validate a CMDX file
cmdx validate docs/api.cmdx

# Verify lossless round-trip
cmdx roundtrip docs/api.md
```

---

## Verifying Correctness

The single most important property of this system:

> **Round-trip invariant:** Decoding an encoded document must produce semantically identical Markdown to the original input.

"Semantically identical" means the Markdown AST is the same, even if surface syntax differs (e.g., `# Heading` vs `Heading\n===`, or different amounts of whitespace between sections). The spec describes a `NormalizeMarkdown` function that re-renders both the original and decoded output through goldmark to produce canonical Markdown for comparison.

**Test this constantly.** Every new feature, every refactor — run the round-trip tests. If they break, the encoder or decoder has a bug.

---

## Understanding the Compression

CMDX achieves savings from three independent mechanisms. They're listed here in order of impact:

1. **Dictionary deduplication (15–40% savings):** Repeated strings are stored once in a `@DICT{}` header and referenced as `$0`, `$1`, etc. This is the biggest win for API docs and technical references, which repeat URLs, type names, and phrases extensively.

2. **Domain-specific blocks (10–20% savings):** Verbose Markdown tables for parameters, fields, and endpoints are collapsed into compact `@KV{}`, `@PARAMS{}`, and `@ENDPOINT{}` blocks. A 7-line Markdown table becomes 3–4 lines of CMDX.

3. **Structural cleanup (5–10% savings):** Removing unnecessary whitespace, collapsing verbose Markdown syntax, and using compact tag notation. Individual savings are small but they add up across a large document.

Total typical savings: **20–35% token reduction** on technical documentation. Prose-heavy content with little repetition will see lower savings (~10–15%).

---

## Dependencies

| Package                       | Version | Purpose                                                                                                                                                                  |
| ----------------------------- | ------- | ------------------------------------------------------------------------------------------------------------------------------------------------------------------------ |
| `github.com/yuin/goldmark`    | v1.7.0  | Markdown parsing. Supports CommonMark and GFM (tables, strikethrough, task lists). The encoder uses its AST walker to convert Markdown into the internal representation. |
| `github.com/stretchr/testify` | v1.9.0  | Test assertions (`require.NoError`, `assert.Equal`). Optional — you can use stdlib `testing` if you prefer.                                                              |

No other dependencies are needed. The dictionary builder, tag parser, serializer, and CLI are all pure Go.

---

## Troubleshooting

**`go mod tidy` fails or downloads unexpected versions:** Delete `go.sum` and run `go mod tidy` again. The sum file will be regenerated from scratch.

**Round-trip test fails but the output "looks right":** The most common cause is whitespace differences. Check for trailing spaces, inconsistent blank lines between sections, or different line endings (LF vs CRLF). The `NormalizeMarkdown` function should handle most of these, but it needs to be implemented carefully.

**Dictionary builder produces different entries on different runs:** The builder must use stable sorting. If two candidates have the same score, break ties by the position of their first occurrence in the document. If you're using `sort.Slice`, switch to `sort.SliceStable`.

**Code blocks contain `@` characters and get parsed as tags:** The parser must treat everything between `@CODE` and `@/CODE` as literal content with no tag processing. If your parser is applying tag expansion inside code blocks, add an "in code block" flag to skip it.

**Large documents cause slow dictionary building:** The candidate extraction step can be O(n²) for large inputs. Limit the sliding window to substrings of 100 characters or fewer, and consider sampling for documents over 100KB. The spec discusses this in the "Edge Cases" section.
