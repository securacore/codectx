## Phase 6: CLI Interface

### Commands

**`codectx init`**
Initialize a new documentation package in the current directory. Defaults to `docs/` as the root directory. If `docs/` already exists and contains non-codectx content, detects the conflict and prompts the user to choose an alternative root (e.g., `ai-docs/`, `.codectx-docs/`, or a custom path). The chosen root is stored in the `root` field of codectx.yaml. Creates the directory structure with all standard directories, default `codectx.yaml`, `ai.yaml`, `preferences.yaml`, default `system/` documentation with sensible compilation instructions, and `.gitignore` additions.

**CLI discovery**: All codectx commands locate the project by walking up from the current directory looking for a codectx.yaml file (similar to how git finds `.git/`). The `root` field in codectx.yaml determines where all documentation and tooling state lives. All path references throughout this specification are relative to whatever root is configured.

**`codectx compile`**
Run the full compilation pipeline. Supports `--incremental` flag (default: true) to only reprocess changed files. Generates `.codectx/compiled/heuristics.yaml` with full diagnostic report. Prints a summary to stdout:

```
Compiled: 342 files → 4,850 chunks (2,134,500 tokens)
Taxonomy: 12,847 terms, 48,203 aliases
Session: 28,450 / 30,000 tokens (94.8%)
Changes: 3 new, 7 modified, 332 unchanged
Time: 47.3s (22.1s LLM augmentation)
```

**`codectx sync`**
Regenerate CLAUDE.md / AGENTS.md / .cursorrules from the current compiled state. Run automatically at the end of `codectx compile`, or standalone after changing `ai.yaml`.

**`codectx query "<search terms>"`**
Search the compiled documentation. Searches all three BM25 indexes (objects, specs, system) and returns results grouped by type. Supports `--top N` flag to override the default `results_count` from ai.yaml.

Internally:
1. Loads taxonomy and translates query terms
2. Runs expanded query against all three BM25 indexes (objects, specs, system)
3. Returns ranked results grouped by type with manifest metadata

Output format (returned to the AI):
```
Results for: "jwt refresh token validation"
Expanded: "jwt json-web-token bearer-token refresh-token token-validation signature-verification"

Instructions:
1. [score: 8.42] obj:a1b2c3.03 — Authentication > JWT Tokens > Refresh Flow
   Source: docs/topics/authentication/jwt-tokens.md (chunk 3/7, 462 tokens)

2. [score: 7.18] obj:a1b2c3.04 — Authentication > JWT Tokens > Validation Rules
   Source: docs/topics/authentication/jwt-tokens.md (chunk 4/7, 488 tokens)

3. [score: 5.91] obj:d4e5f6.02 — Token Service > Refresh Implementation
   Source: docs/topics/token-service/README.md (chunk 2/4, 445 tokens)

Reasoning:
1. [score: 6.85] spec:f7g8h9.02 — Authentication > JWT Tokens > Refresh Flow
   Source: docs/topics/authentication/jwt-tokens.spec.md (chunk 2/3, 380 tokens)

2. [score: 4.21] spec:j1k2l3.01 — Token Validation Strategy
   Source: docs/foundation/error-handling/README.spec.md (chunk 1/2, 290 tokens)

System:
1. [score: 3.12] sys:m3n4o5.01 — Taxonomy Alias Generation Instructions > Rules
   Source: system/topics/taxonomy-generation/README.md (chunk 1/2, 340 tokens)

Related chunks (adjacent to top instruction results, not scored):
  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure
  obj:a1b2c3.05 — Authentication > JWT Tokens > Error Handling
```

**`codectx generate "<chunk-id>,<chunk-id>,<chunk-id>"`**
Assemble specific chunks into a single coherent reading document. Accepts `obj:`, `spec:`, and `sys:` prefixed chunk IDs in a single call, assembling a unified document with clear type demarcation.

Internally:
1. Load requested chunks (from objects/, specs/, or system/ based on prefix)
2. Sort into natural sequence order (even if requested out of order)
3. Group by type: instruction content first, system content second, reasoning content last
4. Restore heading hierarchy for document coherence
5. Insert bridge summaries at boundaries where chunks from the same file are non-adjacent
6. Format according to `output_format` in `ai.yaml`
7. Write to `/tmp/codectx/[topic-slug].[timestamp].md`
8. Tokenize and report count
9. Append footer listing related chunks not requested but adjacent to those that were

Output (returned to the AI):
```
Generated: /tmp/codectx/authentication-jwt-refresh.1741532400.md (1,772 tokens)
Contains: obj:a1b2c3.03, obj:a1b2c3.04, obj:d4e5f6.02, spec:f7g8h9.02

Related chunks not included:
  obj:a1b2c3.02 — Authentication > JWT Tokens > Token Structure (488 tokens)
  obj:a1b2c3.05 — Authentication > JWT Tokens > Error Handling (412 tokens)
  spec:f7g8h9.01 — Authentication > JWT Tokens (380 tokens)
```

**`codectx install`**
Install packages declared in `codectx.yaml`. Resolves `[name]@[org]` references to `github.com/[org]/codectx-[name]`, resolves version tags, clones or fetches to `.codectx/packages/`. If `codectx.lock` exists and `codectx.yaml` hasn't changed, installs from the lock file using pinned commit SHAs (fast, deterministic). If `codectx.yaml` changed, re-resolves affected entries and updates the lock. Resolves transitive dependencies.

**`codectx update`**
Re-resolve all dependencies to their latest compatible versions, update `codectx.lock`, download any changed packages, and recompile if package content changed. This is the command for pulling in newer versions of dependencies — the user should never need flags on `install` for this.

Output:
```
Resolving dependencies...
  react-patterns@community: 2.3.1 → 2.4.0 (updated)
    → github.com/community/codectx-react-patterns@v2.4.0
  company-standards@acme: 2.0.0 (unchanged)
  tailwind-guide@designteam: 2.1.0 (unchanged)
  javascript-fundamentals@community: 1.3.0 (transitive, unchanged)

Updated codectx.lock
Downloaded: react-patterns@community:2.4.0

Recompiling (1 package changed)...
Compiled: 348 files → 4,803 chunks (2,178,200 tokens)
Taxonomy: 12,921 terms, 48,890 aliases
Session: 28,450 / 30,000 tokens (94.8%)
Time: 31.2s
```

**`codectx search "<query>"`**
Search for packages on GitHub using the `codectx-*` naming convention via the GitHub API.

Output:
```
Search results for: "react patterns"

1. react-patterns@community (v2.4.0) ★ 342
   github.com/community/codectx-react-patterns
   React component and hook patterns for AI-driven development

2. react-testing@community (v1.1.0) ★ 89
   github.com/community/codectx-react-testing
   Testing patterns and strategies for React applications

3. react-nextjs@webteam (v3.0.2) ★ 156
   github.com/webteam/codectx-react-nextjs
   Next.js and React integration patterns

Install with: codectx install react-patterns@community:latest
```

**`codectx publish`**
Publish the current package to GitHub. Reads codectx.yaml for name, org, and version. Validates directory structure. Tags the current commit as `v[version]` and pushes to `github.com/[org]/codectx-[name]`. The repo must already exist on GitHub — `codectx publish` handles tagging and validation, not repo creation.

**`codectx session add <reference>`**
Add a local path or package reference to the always-loaded session context. Modifies `session.always_loaded` in `codectx.yaml`. Reports the token cost of the added entry and the new total against the budget.

**`codectx session remove <reference>`**
Remove an entry from the always-loaded session context.

**`codectx session list`**
List all always-loaded session context entries with individual token counts and total against the budget.

**`codectx plan status [plan-name]`**
Report the current state of a plan without loading context. Reads `plan.yaml` and returns the current step, completion percentage, blockers, dependency hash status (changed or unchanged), and stored queries for the current step.

Output:
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)
Progress: 2 steps completed, 1 in progress, 2 pending

Current step: Implement token service refactor
  Started: 2025-03-07T09:00:00Z
  Notes: User service and payment service updated. Order service remaining.
  Stored queries:
    - "token service refactor implementation"
    - "order service authentication"

Dependencies:
  ✓ foundation/architecture-principles — unchanged
  ⚠ topics/authentication/jwt-tokens — content changed since last update
  ✓ topics/authentication/oauth — unchanged

Blocked steps:
  Step 4 (Migration testing) — blocked by step 3
  Step 5 (Production rollout) — blocked by step 4
```

**`codectx plan resume [plan-name]`**
Resume a plan by reconstructing its context. Checks dependency hashes against current compiled state. If all hashes match, replays the current step's stored chunks via `codectx generate` for instant context reconstruction. If hashes changed, reports which dependencies drifted and provides the stored queries for the AI to re-run against updated documentation. Returns the assembled context plus plan state.

Output (hashes match — instant replay):
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)
Dependencies: all unchanged ✓

Replaying context for step 3...
Generated: /tmp/codectx/auth-migration-step3.1741532400.md (1,847 tokens)
Contains: obj:a1b2c3.04, obj:d4e5f6.02, obj:d4e5f6.03, obj:x9y8z7.01, spec:x9y8z7.01

Current step: Implement token service refactor
Notes: User service and payment service updated. Order service remaining.
```

Output (hashes changed — guided re-search):
```
Plan: Authentication System Migration
Status: in-progress (step 3 of 5)

Documentation changes since last update:
  ⚠ topics/authentication/jwt-tokens — content changed
  ✓ foundation/architecture-principles — unchanged
  ✓ topics/authentication/oauth — unchanged

Stored chunks may be stale. Stored queries for current step:
  - "token service refactor implementation"
  - "order service authentication"

Recommendation: Re-run stored queries to refresh context with updated documentation.
```

### Generated File Format

When `codectx generate` assembles chunks, the output is a coherent reading document with content grouped by type: Instructions first, then System (if any `sys:` chunks requested), then Reasoning. Each section is clearly demarcated so the AI understands what is actionable, what is tooling context, and what is explanatory reasoning:

```markdown
<!-- codectx:generated
chunks: obj:a1b2c3.03, obj:a1b2c3.04, obj:d4e5f6.02, spec:f7g8h9.02
sources:
  - docs/topics/authentication/jwt-tokens.md
  - docs/topics/authentication/jwt-tokens.spec.md
  - docs/topics/token-service/README.md
tokens: 1772
generated: 2025-03-09T12:00:00Z
-->

# Instructions

## Authentication > JWT Tokens

### Refresh Flow

The refresh token lifecycle begins when...
[chunk obj:a1b2c3.03 content]

### Validation Rules

Token validation follows a strict sequence...
[chunk obj:a1b2c3.04 content]

---
> **Context bridge**: The above sections covered JWT token management from the
> authentication documentation. The following section is from a different
> document (Token Service) that implements these patterns.
---

## Token Service

### Refresh Implementation

The token service exposes a refresh endpoint...
[chunk obj:d4e5f6.02 content]

---

# Reasoning

> The following sections contain the reasoning behind the instructions above.
> This is informational context for understanding *why* decisions were made.
> Reason about this content before acting on it.

## Authentication > JWT Tokens > Refresh Flow

The refresh token lifecycle was designed around the constraint that...
[chunk spec:f7g8h9.02 content]

---
<!-- codectx:related
Adjacent chunks not included in this document:
- obj:a1b2c3.02: "Authentication > JWT Tokens > Token Structure" (488 tokens)
- obj:a1b2c3.05: "Authentication > JWT Tokens > Error Handling" (412 tokens)
- spec:f7g8h9.01: "Authentication > JWT Tokens" (380 tokens)
Use: codectx generate "obj:a1b2c3.02,obj:a1b2c3.05,spec:f7g8h9.01" to load these
-->
```

---

