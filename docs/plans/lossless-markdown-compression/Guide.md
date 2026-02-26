# CMDX Project — Claude Code Guide

## What is this project?

CMDX (Compressed Markdown Exchange) is a lossless, text-based compression format for Markdown optimized for AI/LLM token consumption. It compresses markdown into a compact text format that LLMs can read directly without decompression.

## Key files

- `CMDX_SPECIFICATION.md` — **Read this first.** Contains the complete format spec, Go architecture, type definitions, algorithm descriptions, implementation order, sample I/O, and design rationale. This is the source of truth.
- `go.mod` — Module definition with dependencies (goldmark for markdown parsing, testify for tests)
- `testdata/api_docs.md` — Sample markdown input fixture
- `testdata/api_docs.cmdx` — Expected compressed output for the above

## Implementation order

Follow the phased approach in the spec doc:

1. **Phase 1 (MVP):** `ast.go` → `escape.go` → `tags.go` → `encoder.go` (passes 1+4) → `decoder.go` → round-trip tests
2. **Phase 2 (Dictionary):** `dict.go` → encoder pass 3 → decoder pass 2 → dictionary tests
3. **Phase 3 (Domain blocks):** encoder pass 2 (KV/PARAMS/ENDPOINT detection) → decoder pass 3 → domain block tests
4. **Phase 4 (CLI):** `cmd/cmdx/main.go` → stats → fuzz tests

## Critical invariant

`NormalizeMarkdown(Decode(Encode(input))) == NormalizeMarkdown(input)` must always hold. Every change should be validated with round-trip tests.

## Architecture

```
Encode: Markdown → (goldmark AST) → Internal AST → Domain Detection → Dictionary Build → Serialize CMDX
Decode: CMDX → Parse Tags → Internal AST → Expand Dictionary → Convert Domain Blocks → Serialize Markdown
```

## Dependencies

- `github.com/yuin/goldmark` — Markdown parser (CommonMark + GFM)
- `github.com/stretchr/testify` — Test assertions

## Conventions

- All source files in package `cmdx` at the repo root
- CLI in `cmd/cmdx/`
- Test fixtures in `testdata/`
- Use table-driven tests
- Fuzz tests for round-trip safety
