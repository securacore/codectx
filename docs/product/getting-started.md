# Getting Started

This guide walks you through installing codectx, initializing a documentation project, writing your first docs, compiling them, and using the search and retrieval system.

---

## Install

**Shell script** (Linux and macOS):

```bash
curl -fsSL https://raw.githubusercontent.com/securacore/codectx/main/bin/install | bash
```

**From source** (requires Go 1.23+):

```bash
go install github.com/securacore/codectx@latest
```

Verify the installation:

```bash
codectx version
```

---

## Initialize a Project

Navigate to your project root and run:

```bash
codectx init
```

This creates the documentation structure under `docs/` (configurable if `docs/` is already in use). The scaffolded structure includes:

```
docs/
  foundation/          # Technology-agnostic guidance
  topics/              # Technology-specific documentation
  plans/               # Work-in-progress tracking
  prompts/             # Pre-crafted AI instructions
  system/              # Compiler behavior instructions (editable)
  codectx.yml          # Project manifest
  .codectx/
    ai.yml             # AI model configuration
    preferences.yml    # Compiler settings
```

The `system/` directory contains default instructions that govern how the compiler processes your documentation — taxonomy generation rules, bridge summary instructions, and context assembly guidelines. These are yours to read and modify.

---

## Write Documentation

Create your first documentation files using the standard structure. Every topic directory has a `README.md` as its entry point.

**Foundation document** — technology-agnostic guidance:

```bash
mkdir -p docs/foundation/coding-standards
```

Create `docs/foundation/coding-standards/README.md`:

```markdown
# Coding Standards

## Naming Conventions

Use descriptive names that convey purpose. Variable names should read
as noun phrases. Function names should read as verb phrases.

- Functions: `calculateTotal`, `validateInput`, `sendNotification`
- Variables: `userCount`, `maxRetries`, `connectionTimeout`
- Constants: `MAX_RETRY_COUNT`, `DEFAULT_TIMEOUT_MS`

## Error Handling

Always use structured error types. Never return raw strings as errors.
Wrap errors with context about what operation failed.
```

**Topic document** — technology-specific:

```bash
mkdir -p docs/topics/authentication
```

Create `docs/topics/authentication/README.md`:

```markdown
# Authentication

## JWT Token Management

The application uses JWT tokens for stateless authentication.
Access tokens expire after 15 minutes. Refresh tokens expire after 7 days.

### Token Validation

Every API request must validate the JWT signature using RS256.
Check the `exp` claim before processing. Reject expired tokens
with a 401 status code.

### Refresh Flow

When an access token expires, the client sends the refresh token
to `/auth/refresh`. The server validates the refresh token,
issues a new access token, and rotates the refresh token.
```

Optionally, add a `.spec.md` file to capture the *reasoning*:

Create `docs/topics/authentication/README.spec.md`:

```markdown
# Authentication Reasoning

## Why JWT Over Session Tokens

JWT was chosen for stateless authentication to avoid server-side
session storage. This enables horizontal scaling without sticky
sessions or shared session stores.

## Why 15-Minute Access Token Expiry

The short expiry window limits the blast radius of a compromised
token. Combined with refresh token rotation, this provides a
balance between security and user experience.
```

---

## Compile

Run the compilation pipeline:

```bash
codectx compile
```

Output:

```
✓ Compilation complete
  Compiled: 3 files -> 8 chunks (3,420 tokens)
  CLI codectx prompt | default search budget: 1,800 tokens (450 × 4 × 1.0)
  Taxonomy: 24 terms extracted
  Session: 0 / 30,000 tokens (0.0%)
  Changes: 3 new, 0 modified, 0 unchanged
  Time: 2.1s
  Output: docs/.codectx/compiled
```

The compiler has:
1. Parsed your markdown into ASTs
2. Stripped unnecessary formatting
3. Split content into token-counted semantic chunks
4. Built a BM25F search index over the chunks
5. Extracted a taxonomy of terms with aliases
6. Generated manifests with chunk metadata and relationships
7. Created compilation heuristics and diagnostics

All compiled output lives in `docs/.codectx/compiled/` (gitignored).

---

## Search

Query your compiled documentation:

```bash
codectx query "token validation"
```

Output:

```
-> Results for: "token validation"
  Expanded: token validation jwt bearer-token

Results (3, bm25f + rrf)
  1. [score: 0.0234] obj:a1b2c3.02 — Authentication > JWT Token Management > Token Validation
     Source: docs/topics/authentication/README.md (chunk 2/3, 312 tokens)
     Indexes: objects:#1

  2. [score: 0.0198] obj:a1b2c3.03 — Authentication > JWT Token Management > Refresh Flow
     Source: docs/topics/authentication/README.md (chunk 3/3, 284 tokens)
     Indexes: objects:#2

  3. [score: 0.0165] spec:d4e5f6.01 — Authentication Reasoning > Why JWT Over Session Tokens
     Source: docs/topics/authentication/README.spec.md (chunk 1/2, 198 tokens)
     Indexes: specs:#1

  Total: 794 tokens across 3 results

Related chunks (adjacent to top results, not scored):
  obj:a1b2c3.01 — Authentication > JWT Token Management

Run "codectx generate" with the top chunk IDs above to read their full content.
```

The search:
- Expanded "token" to include taxonomy aliases like "jwt" and "bearer-token"
- Scored results using BM25F with field-weighted scoring across all three indexes
- Fused the rankings via Reciprocal Rank Fusion into a single interleaved list
- Included adjacent chunks as related context

Use `--top N` to control how many results are returned:

```bash
codectx query --top 5 "error handling patterns"
```

---

## Generate

Assemble specific chunks into a readable document:

```bash
codectx generate "obj:a1b2c3.02,obj:a1b2c3.03,spec:d4e5f6.01"
```

The generated document appears on stdout with content grouped by type — instructions first, reasoning second — with heading hierarchies restored and bridge summaries at chunk boundaries.

The summary appears on stderr:

```
✓ Generated (794 tokens, hash: e7f8a9b0c1d2)
  History: .codectx/history/docs/1741532401000000000.e7f8a9b0c1d2.md
  Contains: obj:a1b2c3.02, obj:a1b2c3.03, spec:d4e5f6.01
  Related chunks not included:
    obj:a1b2c3.01 — Authentication > JWT Token Management (246 tokens)
```

Write to a file instead:

```bash
codectx generate --file output.md "obj:a1b2c3.02,obj:a1b2c3.03"
```

---

## One-Step Query and Generate

The `prompt` command combines query and generate into a single atomic operation — search for relevant chunks and immediately assemble the top results into a reading document:

```bash
codectx prompt "jwt token refresh flow"
```

This is the fastest path from question to assembled documentation. It accepts the same `--top`, `--file`, and `--no-cache` flags as the individual commands, plus `--budget` to set a token ceiling for the assembled output.

---

## Set Up AI Tool Integration

Generate entry point files that tell AI tools how to find and use your compiled documentation:

```bash
codectx link
```

This creates entry point files (CLAUDE.md, AGENTS.md, .cursorrules, copilot-instructions.md) that direct the AI to read your compiled session context and use `codectx query` and `codectx generate` for documentation access.

---

## Configure Session Context

Add your most important documentation to the always-loaded session context — the document the AI reads at the start of every session:

```bash
codectx session add foundation/coding-standards
codectx session list
```

Output:

```
-> Always-loaded session context (1,840 / 30,000 tokens (6.1%))

  foundation/coding-standards    1,840 tokens
```

The session context is compiled into `docs/.codectx/compiled/context.md` on the next `codectx compile`. See [Session Context](session-context.md) for details on budgets and package references.

---

## Next Steps

- [How It Works](how-it-works.md) — Understand the compilation pipeline in detail
- [Documentation Structure](documentation-structure.md) — Learn the content types and conventions
- [Search and Retrieval](search-and-retrieval.md) — How the search pipeline scores and ranks results
- [Configuration](configuration.md) — Customize compilation, search, and AI model settings
- [Package Manager](package-manager.md) — Install and share documentation packages
- [CLI Reference](cli-reference.md) — Complete command reference with all flags

[Back to overview](README.md)
