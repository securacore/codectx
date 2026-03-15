# History and Caching

codectx tracks every query and every generated document, provides a deterministic cache for the generate pipeline, and maintains usage metrics at both local and project-wide scopes.

---

## History

### What Gets Tracked

Every `codectx query` invocation creates a record in `history/queries/`:
- The raw query string
- The taxonomy-expanded query
- Result count
- Compilation state hash (for staleness detection)
- Caller context (which AI tool, session ID, model)

Every `codectx generate` invocation creates a paired record — a metadata entry in `history/chunks/` and the actual document in `history/docs/`:
- Chunk IDs that were assembled
- Token count of the generated document
- Content hash
- Compilation state hash
- Whether it was served from cache
- Caller context

### Viewing History

```bash
codectx history queries
```

```
Recent queries (last 10):

  1. jwt authentication                               30 results   2m ago
     Expanded: jwt authentication bearer-token authn oauth
     Caller:   claude  Session: sess_abc123
     Compile:  x7y8z9 (current)

  2. error handling middleware                        28 results  14m ago
     Expanded: error handling middleware error-boundary
     Caller:   claude  Session: sess_abc123
     Compile:  x7y8z9 (current)

  3. database connection pooling                      30 results   3d ago
     Expanded: database connection pooling db pool
     Caller:   cursor  Session: unknown
     Compile:  a1b2c3 (stale -- recompiled since this query)
```

```bash
codectx history chunks
```

```
Recent generates (last 10):

  1. a1b2c3d4e5f6   1,772 tokens   2m ago
     obj:a1b2c3.01, obj:a1b2c3.02, obj:d4e5f6.02, spec:f7g8h9.01
     Caller: claude  Session: sess_abc123  [from cache: no]
     Compile: x7y8z9 (current)

  2. b2c3d4e5f6a1   2,104 tokens  14m ago
     obj:b2c3d4.01, obj:b2c3d4.02, obj:b2c3d4.03
     Caller: claude  Session: sess_abc123  [from cache: yes]
     Compile: x7y8z9 (current)
```

### Retrieving Past Documents

```bash
codectx history show a1b2c3
```

Prefix-matches against content hashes in `history/docs/`. If multiple matches exist, the newest is returned.

### Clearing History

```bash
codectx history clear
```

Requires interactive terminal confirmation. Deletes all files in `history/queries/`, `history/chunks/`, and `history/docs/`. Does not affect usage metrics.

---

## Generate Cache

The generate pipeline caches its output. When the same chunk IDs are requested against the same compilation state, the cached document is returned instantly without re-assembling.

### Cache Key

Two fields must match for a cache hit:

| Field | Value |
|-------|-------|
| `chunk_set_hash` | SHA-256 of sorted chunk IDs, comma-joined |
| `compile_hash` | SHA-256 of `hashes.yml` file content |

Sorting chunk IDs before hashing ensures order doesn't affect the cache key — requesting chunks in any order hits the same cache entry. The compile hash ensures that if documentation has been recompiled (even without changing chunk IDs), the cache is invalidated. Chunk content, bridge summaries, or heading paths may have changed.

### Cache Hit Behavior

On a cache hit, the generate pipeline skips all assembly steps. The cached document is read from disk and printed to stdout. Output is identical to a normal invocation. The `[from cache]` annotation appears only in the summary:

```
-> Generated (1,772 tokens, hash: a1b2c3d4e5f6) [from cache]
  History: .codectx/history/docs/1741532401000000000.a1b2c3d4e5f6.md
  Contains: obj:a1b2c3.01, obj:a1b2c3.02, obj:d4e5f6.02, spec:f7g8h9.01
```

### Bypassing Cache

```bash
codectx generate --no-cache "obj:a1b2c3.01,obj:a1b2c3.02"
```

Forces the full generate pipeline to run regardless of cache state. The fresh result is still written to history for future cache lookups.

### Cache Self-Healing

If a cached document has been pruned from `history/docs/` but the `chunks/` entry still references it, the cache lookup detects the missing file and treats it as a miss. The generate pipeline runs normally, producing a fresh result and repopulating the cache. No manual intervention needed.

---

## Caller Context Detection

Every invocation automatically detects which AI tool is calling codectx. This metadata is recorded in history entries and usage metrics.

### Detection Priority

1. **Explicit environment variables**: `CODECTX_CALLER`, `CODECTX_SESSION_ID`, `CODECTX_MODEL`
2. **Tool-specific environment variables**: `CLAUDE_CODE_ENTRYPOINT`, `CLAUDE_CODE_SESSION_ID`, `CURSOR_SESSION_ID`, `ANTHROPIC_MODEL`
3. **Parent process name**: Detected via system process inspection
4. **Fallback**: `"unknown"`

Caller detection is best-effort. Detection errors are never surfaced to the user.

---

## Usage Metrics

Two files track invocation and token usage at different scopes.

### Local Usage (per-machine)

`usage.yml` is gitignored and updated on every query and generate invocation:

```yaml
total_tokens: 821903
query_invocations: 347
generate_invocations: 89
cache_hits: 41

tokens_by_caller:
  claude: 601450
  cursor: 198203
  unknown: 22250

tokens_by_model:
  claude-sonnet-4-20250514: 601450
  unknown: 220453
```

### Global Usage (project lifetime)

`global_usage.yml` is checked into version control and updated only on `codectx compile`. It accumulates from `usage.yml` at compile time, surviving machine changes and team contributions.

```yaml
total_tokens: 5621903
query_invocations: 2847
generate_invocations: 612
cache_hits: 298

tokens_by_caller:
  claude: 4201450
  cursor: 1220203
  unknown: 200250
```

### Viewing Usage

```bash
codectx usage
```

```
Token usage (local machine):

  Total tokens generated:      821,903
  Query invocations:               347
  Generate invocations:             89
  Cache hit rate:               46.1%  (41 / 89)

  By caller:
    claude                     601,450 tokens  (73.2%)
    cursor                     198,203 tokens  (24.1%)
    unknown                     22,250 tokens   (2.7%)

  Tracking since: 2025-03-09   Last updated: 2025-03-14
  Last synced to global:       2025-03-13 (on compile)

Project lifetime (global_usage.yml):

  Total tokens generated:    5,621,903
  Query invocations:             2,847
  Generate invocations:            612
  Cache hit rate:               48.7%  (298 / 612)
```

Flags:
- `--local` — show only local machine metrics
- `--global` — show only project lifetime metrics
- `--reset-local` — reset `usage.yml` to zero without syncing to global

### Sync Timing

Local usage is synced to global usage on `codectx compile` — a deliberate, committed act that's already checked into version control. This avoids noisy diffs and frequent merge conflicts that per-invocation syncing would cause.

---

## Pruning

The `history/` directory is capped at 100MB. When the limit is exceeded, pruning retains the 5 most recent files in each subdirectory (`queries/`, `chunks/`, `docs/`). Pruning runs automatically after each generate invocation.

Timestamp-first filenames mean lexicographic sort equals chronological order — pruning simply deletes the oldest entries.

---

## Error Handling

All history, cache, and usage operations are best-effort. Failures warn to stderr and never block the primary command. If a query or generate succeeds but the history write fails, the user gets their results with a warning — not a failure.

---

[Back to overview](README.md)
