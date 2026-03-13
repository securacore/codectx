# codectx — History, Cache & Usage Specification

## Supplement to codectx-system-spec.md — Phase 9 Replacement

**Status**: Approved for implementation
**Supersedes**: Phase 9 (History System) in codectx-system-spec.md, lines 1953–2014
**Affected commands**: `codectx query`, `codectx generate`, `codectx history`, `codectx compile`, `codectx usage`, `codectx init`, `codectx repair`

---

## Overview

This specification replaces the TSV-based history system (Phase 9 of the system spec) with a structured per-file history, deterministic generate caching, caller context tracking, and dual-scope usage metrics. The existing `query.history` and `chunks.history` flat files are eliminated entirely. No migration is provided — this is a clean replacement.

---

## Design Principles

**Best-effort throughout.** History, cache, and usage writes never block or fail query or generate commands. Any write failure warns to stderr and continues.

**Cache miss is never fatal.** A pruned or missing document is a cache miss. The generate pipeline runs, produces a fresh result, and re-populates the cache. The system self-heals without intervention.

**No index, no configuration.** Relationships between history entries are encoded in filenames and file content. Hash-based lookups are filesystem globs. The directory structure is the schema.

**Strict 1:1 chain.** One `codectx query` invocation produces one `queries/` entry. One `codectx generate` invocation produces one `chunks/` entry and one `docs/` entry, always as a pair. The AI's chunk selection is the only non-deterministic link and is intentionally not system-enforced.

**Compile is the global checkpoint.** Local usage accumulates on every invocation. Global lifetime usage is updated only on `codectx compile` — a deliberate act that is already committed to version control.

---

## 1. Directory Structure

```
.codectx/
  history/                                          # Gitignored
    queries/
      [nanoTs].[query_hash12].json                  # One per codectx query invocation
    chunks/
      [nanoTs].[chunk_set_hash12].json              # One per codectx generate invocation
    docs/
      [nanoTs].[content_hash12].md                  # One per codectx generate invocation
  usage.yml                                         # Gitignored — local machine usage metrics
  global_usage.yml                                  # Checked in — project lifetime usage metrics
```

### Filename Format

All history files use the format `[nanoTs].[hash12].[ext]` where:

- `nanoTs` is a fixed-width nanosecond Unix timestamp (19 digits)
- `hash12` is the first 12 hex characters of the relevant SHA-256 hash (after the `sha256:` prefix)
- `ext` is `json` for entries or `md` for documents

Timestamp-first ordering means `sort.Strings()` on a directory listing produces chronological order. This simplifies pruning and cache lookup — no timestamp extraction or numeric parsing is needed.

### Gitignore Rules

The managed gitignore template in `core/project/gitignore.go` is updated to:

```go
var gitignoreTemplate = []string{
    "# Compiled output and tooling state",
    "{{ROOT}}/.codectx/compiled/",
    "{{ROOT}}/.codectx/packages/",
    "{{ROOT}}/.codectx/history/",
    "{{ROOT}}/.codectx/ai.local.yml",
    "{{ROOT}}/.codectx/usage.yml",
    "",
    "# Force-include checked-in config and lifetime metrics",
    "!{{ROOT}}/.codectx/ai.yml",
    "!{{ROOT}}/.codectx/preferences.yml",
    "!{{ROOT}}/.codectx/global_usage.yml",
}
```

---

## 2. Hash Semantics

Each directory uses a distinct hash type. The location makes the hash type self-documenting.

| Directory / File | Hash Field | Hash Of | Purpose |
|------------------|------------|---------|---------|
| `queries/` | `query_hash` | Raw query string | Audit identity |
| `chunks/` | `chunk_set_hash` | Sorted chunk IDs, comma-joined | Generate cache key |
| `docs/` | `content_hash` | Assembled document bytes | Document identity |
| `chunks/` | `compile_hash` | `hashes.yml` file content | Cache invalidation signal |

All hashes are SHA-256, prefixed `sha256:`. Filenames use the first 12 hex characters after the prefix.

---

## 3. File Schemas

### 3.1. queries/[nanoTs].[query_hash12].json

One file written per `codectx query` invocation.

```json
{
  "ts": 1741532400000000000,
  "query_hash": "sha256:a1b2c3d4e5f6...",
  "raw": "jwt authentication",
  "expanded": "jwt authentication bearer-token authn oauth session-auth",
  "result_count": 30,
  "compile_hash": "sha256:x7y8z9...",
  "caller": "claude",
  "session_id": "sess_abc123",
  "model": "claude-sonnet-4-20250514"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ts` | int64 | Nanosecond Unix timestamp |
| `query_hash` | string | `sha256:` + SHA-256 hex of raw query string |
| `raw` | string | Query as typed |
| `expanded` | string | Taxonomy-expanded terms, space-joined |
| `result_count` | int | Total chunks returned across all index types |
| `compile_hash` | string | `sha256:` + SHA-256 hex of `hashes.yml` content at invocation time |
| `caller` | string | Detected calling program (see Caller Context Detection) |
| `session_id` | string | Detected session ID (see Caller Context Detection) |
| `model` | string | Detected model identifier (see Caller Context Detection) |

The `compile_hash` field serves no caching purpose for queries — it is recorded for audit and staleness display in `codectx history queries`. Query result caching is not implemented in this specification.

### 3.2. chunks/[nanoTs].[chunk_set_hash12].json

One file written per `codectx generate` invocation. Always created as a pair with its `docs/` counterpart.

```json
{
  "ts": 1741532401000000000,
  "chunk_set_hash": "sha256:d4e5f6a1b2c3...",
  "chunks": [
    "obj:a1b2c3.01",
    "obj:a1b2c3.02",
    "obj:d4e5f6.02",
    "spec:f7g8h9.01"
  ],
  "token_count": 1772,
  "content_hash": "sha256:a1b2c3d4e5f6...",
  "compile_hash": "sha256:x7y8z9...",
  "doc_file": "1741532401000000000.a1b2c3d4e5f6.md",
  "cache_hit": false,
  "caller": "claude",
  "session_id": "sess_abc123",
  "model": "claude-sonnet-4-20250514"
}
```

| Field | Type | Description |
|-------|------|-------------|
| `ts` | int64 | Nanosecond Unix timestamp |
| `chunk_set_hash` | string | `sha256:` + SHA-256 hex of sorted chunk IDs, comma-joined — primary cache key |
| `chunks` | []string | All chunk IDs passed to this invocation, in the order provided by the caller |
| `token_count` | int | Total tokens in assembled document |
| `content_hash` | string | `sha256:` + SHA-256 hex of assembled document bytes |
| `compile_hash` | string | `sha256:` + SHA-256 hex of `hashes.yml` content — cache invalidation signal |
| `doc_file` | string | Filename of corresponding `docs/` entry |
| `cache_hit` | bool | Whether this invocation was served from cache |
| `caller` | string | Detected calling program |
| `session_id` | string | Detected session ID |
| `model` | string | Detected model identifier |

The `chunks` field preserves the caller-provided order (natural sequence). The `chunk_set_hash` is computed from a sorted, deduplicated copy — order does not affect the cache key.

### 3.3. docs/[nanoTs].[content_hash12].md

One file written per `codectx generate` invocation. Always created as a pair with its `chunks/` counterpart. Content is identical to the document produced by the generate pipeline — the exact bytes written to stdout or `--file`.

No structural changes to the document format. The `<!-- codectx:generated ... -->` header and `<!-- codectx:related ... -->` footer remain as specified in the system spec.

---

## 4. The 1:1 Audit Trail

```
queries/1741532400000000000.a1b2c3d4e5f6.json     <- what was searched
  ts:           1741532400000000000
  query_hash:   sha256:a1b2c3...
  caller:       claude
  session_id:   sess_abc123
  compile_hash: sha256:x7y8z9...
        |
        |  (AI decision -- traceable by session_id + timestamp proximity)
        v
chunks/1741532401000000000.d4e5f6a1b2c3.json      <- what chunks were assembled
  ts:             1741532401000000000
  chunk_set_hash: sha256:d4e5f6...
  compile_hash:   sha256:x7y8z9...
  content_hash:   sha256:a1b2c3...
  doc_file:       1741532401000000000.a1b2c3d4e5f6.md
  caller:         claude
  session_id:     sess_abc123
        |
        |  (explicit content_hash + doc_file reference)
        v
docs/1741532401000000000.a1b2c3d4e5f6.md           <- what document was produced
```

The query-to-chunks link is traceable via shared `session_id` and timestamp proximity. It is not system-enforced — the AI's chunk selection is the non-deterministic step. The chunks-to-docs link is always an explicit reference via `doc_file`.

---

## 5. Caller Context Detection

Resolved at every invocation using a priority chain. Failure at any step falls through to the next. Best-effort — detection errors are never surfaced to the user.

### Environment Variable Contract

```go
const (
    EnvCaller    = "CODECTX_CALLER"
    EnvSessionID = "CODECTX_SESSION_ID"
    EnvModel     = "CODECTX_MODEL"
)
```

These are codectx-defined environment variables. Calling tools may set them to provide explicit context. They take highest priority in the resolution chain.

### Resolution Logic

```go
type CallerContext struct {
    Caller    string
    SessionID string
    Model     string
}

func ResolveCallerContext() CallerContext {
    return CallerContext{
        Caller: coalesce(
            os.Getenv(EnvCaller),
            os.Getenv("CLAUDE_CODE_ENTRYPOINT"),
            detectParentProcessName(),
        ),
        SessionID: coalesce(
            os.Getenv(EnvSessionID),
            os.Getenv("CLAUDE_CODE_SESSION_ID"),
            os.Getenv("CURSOR_SESSION_ID"),
        ),
        Model: coalesce(
            os.Getenv(EnvModel),
            os.Getenv("ANTHROPIC_MODEL"),
        ),
    }
}

func coalesce(vals ...string) string {
    for _, v := range vals {
        if v != "" {
            return v
        }
    }
    return "unknown"
}
```

### Parent Process Detection

Uses `github.com/shirou/gopsutil/v3/process` for cross-platform parent process name detection:

```go
func detectParentProcessName() string {
    parent, err := process.NewProcess(int32(os.Getppid()))
    if err != nil {
        return ""
    }
    name, err := parent.Name()
    if err != nil {
        return ""
    }
    return name
}
```

Returns empty string on any error, allowing `coalesce` to fall through to the next candidate or `"unknown"`.

---

## 6. Generate Cache

### 6.1. Cache Key

Both fields must match for a cache hit:

| Field | Value |
|-------|-------|
| `chunk_set_hash` | `sha256:` + SHA-256 hex of sorted chunk IDs, comma-joined |
| `compile_hash` | `sha256:` + SHA-256 hex of `hashes.yml` file content |

A matching chunk set against a changed compilation is always a cache miss — chunk content, bridge summaries, or heading paths may have changed.

### 6.2. Computing chunk_set_hash

```go
func ChunkSetHash(chunkIDs []string) string {
    sorted := make([]string, len(chunkIDs))
    copy(sorted, chunkIDs)
    sort.Strings(sorted)
    raw := strings.Join(sorted, ",")
    sum := sha256.Sum256([]byte(raw))
    return "sha256:" + hex.EncodeToString(sum[:])
}
```

Sorting before hashing ensures chunk ID order does not affect the cache key.

### 6.3. Computing compile_hash

```go
func CompileHash(compiledDir string) (string, error) {
    hashesPath := manifest.HashesPath(compiledDir)
    data, err := os.ReadFile(hashesPath)
    if err != nil {
        return "", err
    }
    sum := sha256.Sum256(data)
    return "sha256:" + hex.EncodeToString(sum[:]), nil
}
```

Uses `manifest.HashesPath` to resolve the correct path to `hashes.yml` within the compiled directory.

### 6.4. Cache Lookup Flow

```go
func GenerateCacheLookup(historyDir string, chunkIDs []string, compiledDir string) (docPath string, hit bool) {
    chunkSetHash := ChunkSetHash(chunkIDs)
    compileHash, err := CompileHash(compiledDir)
    if err != nil {
        return "", false
    }

    shortHash := chunkSetHash[7:19] // strip "sha256:", take first 12 hex chars
    pattern := filepath.Join(historyDir, "chunks", "*."+shortHash+".json")
    matches, err := filepath.Glob(pattern)
    if err != nil || len(matches) == 0 {
        return "", false
    }

    // Lexicographic descending = newest first (timestamp-prefixed filenames).
    sort.Sort(sort.Reverse(sort.StringSlice(matches)))

    for _, match := range matches {
        entry, err := readChunksEntry(match)
        if err != nil {
            continue
        }

        // Verify full hash matches (not just the 12-char prefix).
        if entry.ChunkSetHash != chunkSetHash {
            continue
        }

        // Verify compilation state matches.
        if entry.CompileHash != compileHash {
            continue
        }

        // Verify the docs/ file still exists (may have been pruned).
        docPath = filepath.Join(historyDir, "docs", entry.DocFile)
        if _, err := os.Stat(docPath); os.IsNotExist(err) {
            continue
        }

        return docPath, true
    }

    return "", false
}
```

Key properties:

- Glob uses `*.[shortHash].json` — the wildcard matches the timestamp prefix.
- Full `chunk_set_hash` is verified inside the loop to guard against 12-character prefix collisions.
- Full `compile_hash` is verified to ensure compilation state matches.
- Doc file existence is verified to handle pruned documents gracefully.
- Newest match is tried first (lexicographic descending on timestamp-prefixed filenames).

### 6.5. Updated Generate Pipeline

Cache lookup inserts as step 0, before any assembly:

```
Step 0:  Cache lookup (skipped if --no-cache flag is set)
         -> compute chunk_set_hash and compile_hash
         -> glob history/chunks/*.[chunk_set_hash12].json
         -> for each match (newest first): verify full hash, check compile_hash, verify doc exists
         -> HIT:  read doc from disk, print to stdout, print summary with [from cache], return
         -> MISS: continue to step 1

Step 1:  Load requested chunks from compiled output
Step 2:  Sort into natural sequence order
Step 3:  Group by type (instructions -> system -> reasoning)
Step 4:  Restore heading hierarchy
Step 5:  Insert bridge summaries at non-adjacent boundaries
Step 6:  Format per output_format in ai.yml
Step 7:  Compute SHA-256 content hash
Step 8:  Print document to stdout (or --file)
Step 9:  Print summary to stderr
Step 10: Write chunks/ entry and docs/ entry as a pair (best-effort)
Step 11: Update usage.yml (best-effort)
```

On a cache hit, steps 1-9 are skipped entirely. A chunks/ entry is still written (with `cache_hit: true`) and usage metrics are still updated.

### 6.6. Cache Hit Output

Output is identical to a normal generate invocation. The `[from cache]` annotation appears only in the summary line:

```
[document content on stdout -- identical to original]

# On stderr:
-> Generated (1,772 tokens, hash: a1b2c3d4e5f6) [from cache]
  History: .codectx/history/docs/1741532401000000000.a1b2c3d4e5f6.md
  Contains: obj:a1b2c3.01, obj:a1b2c3.02, obj:d4e5f6.02, spec:f7g8h9.01
  Related chunks not included:
    obj:a1b2c3.02 -- Authentication > JWT Tokens > Token Structure (488 tokens)
    obj:a1b2c3.05 -- Authentication > JWT Tokens > Error Handling (412 tokens)
```

### 6.7. --no-cache Flag

```go
&cli.BoolFlag{
    Name:  "no-cache",
    Usage: "Bypass cache lookup and always run the full generate pipeline",
}
```

When set, step 0 is skipped entirely. The generate pipeline runs from step 1 as if no cache exists. The result is still written to history (creating or replacing cache entries for future lookups).

---

## 7. Usage Metrics

Two YAML files track invocation and token usage at different scopes.

### 7.1. usage.yml (gitignored -- local machine)

Updated on every `codectx query` and `codectx generate` invocation. Read-modify-write, best-effort. Represents usage on this machine only.

```yaml
# .codectx/usage.yml
# Local machine usage -- gitignored. Rolled into global_usage.yml on codectx compile.

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

first_seen: 1741532400000000000
last_updated: 1741998712000000000
```

### 7.2. global_usage.yml (checked in -- project lifetime)

Updated only on `codectx compile`. Accumulates from `usage.yml` at compile time. Survives machine changes, team contributions, and history pruning. This is the lifetime record.

```yaml
# .codectx/global_usage.yml
# Project lifetime usage -- checked into version control.
# Updated on codectx compile from local usage.yml.

total_tokens: 5621903
query_invocations: 2847
generate_invocations: 612
cache_hits: 298

tokens_by_caller:
  claude: 4201450
  cursor: 1220203
  unknown: 200250

tokens_by_model:
  claude-sonnet-4-20250514: 4201450
  unknown: 1420453

project: my-project
first_seen: 1741532400000000000
last_updated: 1741998712000000000
last_compile: 1741998712000000000
```

On multi-developer teams, `global_usage.yml` is checked into version control. Concurrent modifications by different developers will produce merge conflicts on integer counters — these are resolved manually. This is an accepted trade-off for single-file simplicity.

### 7.3. Update Logic

**On query invocation** — increment counters in `usage.yml`:

```go
func UpdateUsageQuery(usageFile string, caller CallerContext) {
    usage := readOrInitUsage(usageFile)
    usage.QueryInvocations++
    usage.LastUpdated = time.Now().UnixNano()
    writeUsage(usageFile, usage)
}
```

Query invocations do not produce tokens. Only the invocation counter is incremented.

**On generate invocation** — add tokens and increment counters in `usage.yml`:

```go
func UpdateUsageGenerate(usageFile string, tokens int, cacheHit bool, caller CallerContext) {
    usage := readOrInitUsage(usageFile)
    usage.TotalTokens += tokens
    usage.GenerateInvocations++
    if cacheHit {
        usage.CacheHits++
    }
    usage.TokensByCaller[caller.Caller] += tokens
    usage.TokensByModel[caller.Model] += tokens
    usage.LastUpdated = time.Now().UnixNano()
    writeUsage(usageFile, usage)
}
```

**On `codectx compile`** — merge `usage.yml` into `global_usage.yml`, then reset local:

```go
func SyncGlobalUsage(localFile, globalFile, projectName string) error {
    local := readOrInitUsage(localFile)
    global := readOrInitUsage(globalFile)

    global.TotalTokens += local.TotalTokens
    global.QueryInvocations += local.QueryInvocations
    global.GenerateInvocations += local.GenerateInvocations
    global.CacheHits += local.CacheHits

    for caller, tokens := range local.TokensByCaller {
        global.TokensByCaller[caller] += tokens
    }
    for model, tokens := range local.TokensByModel {
        global.TokensByModel[model] += tokens
    }

    global.Project = projectName
    global.LastUpdated = time.Now().UnixNano()
    global.LastCompile = time.Now().UnixNano()

    if err := writeUsage(globalFile, global); err != nil {
        return err
    }
    return writeUsage(localFile, initUsage())
}
```

Local usage resets to zero after a successful sync. If the global write succeeds but the local reset fails, the next compile will re-add the same local counts — a slight overcount, acceptable for a volume measurement.

### 7.4. Race Conditions

Concurrent `codectx` invocations on the same project (e.g., two AI sessions) may race on `usage.yml` read-modify-write. One write may be lost. This is acceptable — usage metrics are approximate volume measurements, not financial accounting. No file locking is used.

---

## 8. Pruning

**Threshold**: 100MB total across the `history/` directory.
**Retention**: Last 5 files per subdirectory (`queries/`, `chunks/`, `docs/`).
**Trigger**: After every `codectx generate` invocation, via `CheckAndPrune`.

```go
func PruneDirectory(dir string, keepN int) error {
    entries, err := filepath.Glob(filepath.Join(dir, "*"))
    if err != nil || len(entries) <= keepN {
        return err
    }

    // Timestamp-first filenames: lexicographic sort = chronological order.
    sort.Strings(entries)

    // Delete oldest entries (beginning of sorted list).
    for _, path := range entries[:len(entries)-keepN] {
        os.Remove(path) // best-effort
    }
    return nil
}

func CheckAndPrune(historyDir string) {
    size, _ := dirSize(historyDir)
    if size < MaxSize {
        return
    }
    PruneDirectory(filepath.Join(historyDir, "queries"), PruneKeep)
    PruneDirectory(filepath.Join(historyDir, "chunks"), PruneKeep)
    PruneDirectory(filepath.Join(historyDir, "docs"), PruneKeep)
}
```

### Cache Miss After Pruning

If a `chunks/` entry references a `doc_file` that no longer exists in `docs/`, `GenerateCacheLookup` detects the missing file via `os.Stat` and treats it as a cache miss. The generate pipeline runs normally, writing a fresh pair. No special pruning awareness required.

---

## 9. Go Types

```go
// QueryEntry represents a single query invocation record.
type QueryEntry struct {
    Ts          int64  `json:"ts"`
    QueryHash   string `json:"query_hash"`
    Raw         string `json:"raw"`
    Expanded    string `json:"expanded"`
    ResultCount int    `json:"result_count"`
    CompileHash string `json:"compile_hash"`
    Caller      string `json:"caller"`
    SessionID   string `json:"session_id"`
    Model       string `json:"model"`
}

// ChunksEntry represents a single generate invocation record.
type ChunksEntry struct {
    Ts           int64    `json:"ts"`
    ChunkSetHash string   `json:"chunk_set_hash"`
    Chunks       []string `json:"chunks"`
    TokenCount   int      `json:"token_count"`
    ContentHash  string   `json:"content_hash"`
    CompileHash  string   `json:"compile_hash"`
    DocFile      string   `json:"doc_file"`
    CacheHit     bool     `json:"cache_hit"`
    Caller       string   `json:"caller"`
    SessionID    string   `json:"session_id"`
    Model        string   `json:"model"`
}

// CallerContext holds detected caller metadata.
type CallerContext struct {
    Caller    string
    SessionID string
    Model     string
}

// UsageMetrics tracks invocation and token usage.
type UsageMetrics struct {
    TotalTokens         int            `yaml:"total_tokens"`
    QueryInvocations    int            `yaml:"query_invocations"`
    GenerateInvocations int            `yaml:"generate_invocations"`
    CacheHits           int            `yaml:"cache_hits"`
    TokensByCaller      map[string]int `yaml:"tokens_by_caller"`
    TokensByModel       map[string]int `yaml:"tokens_by_model"`
    Project             string         `yaml:"project,omitempty"`
    FirstSeen           int64          `yaml:"first_seen"`
    LastUpdated         int64          `yaml:"last_updated"`
    LastCompile         int64          `yaml:"last_compile,omitempty"`
}
```

---

## 10. Write Patterns

### 10.1. History Entries (Atomic File Creation)

Each history write creates a new file — no append, no locking, no read-modify-write:

```go
func WriteQueryEntry(historyDir string, entry QueryEntry) error {
    shortHash := entry.QueryHash[7:19] // strip "sha256:", take 12 hex chars
    filename := fmt.Sprintf("%d.%s.json", entry.Ts, shortHash)
    path := filepath.Join(historyDir, "queries", filename)
    data, err := json.MarshalIndent(entry, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, project.FilePerm)
}

func WriteChunksEntry(historyDir string, entry ChunksEntry) error {
    shortHash := entry.ChunkSetHash[7:19]
    filename := fmt.Sprintf("%d.%s.json", entry.Ts, shortHash)
    path := filepath.Join(historyDir, "chunks", filename)
    data, err := json.MarshalIndent(entry, "", "  ")
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, project.FilePerm)
}
```

Uses `json.MarshalIndent` for human-readable entries.

### 10.2. Usage Files (Read-Modify-Write)

```go
func readOrInitUsage(path string) UsageMetrics {
    data, err := os.ReadFile(path)
    if err != nil {
        return UsageMetrics{
            TokensByCaller: map[string]int{},
            TokensByModel:  map[string]int{},
            FirstSeen:      time.Now().UnixNano(),
        }
    }
    var m UsageMetrics
    _ = yaml.Unmarshal(data, &m)
    if m.TokensByCaller == nil {
        m.TokensByCaller = map[string]int{}
    }
    if m.TokensByModel == nil {
        m.TokensByModel = map[string]int{}
    }
    return m
}

func writeUsage(path string, m UsageMetrics) error {
    data, err := yaml.Marshal(m)
    if err != nil {
        return err
    }
    return os.WriteFile(path, data, project.FilePerm)
}
```

### 10.3. Error Handling Convention

All history and usage writes use best-effort semantics:

```go
func bestEffort(op string, err error) {
    if err != nil {
        fmt.Fprintf(os.Stderr, "warning: %s: %v\n", op, err)
    }
}
```

---

## 11. CLI Commands

### 11.1. codectx history queries

Reads `history/queries/`, sorted by timestamp descending. Shows compile staleness by comparing each entry's `compile_hash` against the current `hashes.yml`.

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

### 11.2. codectx history chunks

Reads `history/chunks/`, sorted by timestamp descending.

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

### 11.3. codectx history show <hash>

Prefix-matches against `history/docs/` filenames. Since filenames are `[nanoTs].[hash12].md`, the hash prefix match looks for `*.[hashPrefix]*.md` patterns. If multiple matches exist, the newest (highest timestamp, lexicographically last) is returned.

### 11.4. codectx history clear

Deletes all files in `history/queries/`, `history/chunks/`, and `history/docs/`. Requires interactive terminal confirmation via `huh` prompt. Refuses in non-interactive mode. Does not affect `usage.yml` or `global_usage.yml`.

### 11.5. codectx usage

New command. Displays local and global usage metrics.

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

  By model:
    claude-sonnet-4-20250514   601,450 tokens  (73.2%)
    unknown                    220,453 tokens  (26.8%)

  Tracking since: 2025-03-09   Last updated: 2025-03-14
  Last synced to global:       2025-03-13 (on compile)

Project lifetime (global_usage.yml):

  Total tokens generated:    5,621,903
  Query invocations:             2,847
  Generate invocations:            612
  Cache hit rate:               48.7%  (298 / 612)
  First recorded:            2025-03-09
  Last compile sync:         2025-03-13
```

**Flags:**

| Flag | Description |
|------|-------------|
| `--global` | Print only the global (project lifetime) section |
| `--local` | Print only the local (machine) section |
| `--reset-local` | Reset `usage.yml` to zero without syncing to global (debugging) |

---

## 12. Scaffold and Repair

### 12.1. codectx init

Creates the new directory structure during project initialization:

```
.codectx/history/queries/    (empty)
.codectx/history/chunks/     (empty)
.codectx/history/docs/       (empty)
.codectx/usage.yml           (initialized to zero values)
.codectx/global_usage.yml    (initialized to zero values with project name)
```

### 12.2. codectx repair

Ensures all history subdirectories exist (`queries/`, `chunks/`, `docs/`). Creates missing `usage.yml` or `global_usage.yml` with zero values. Never overwrites existing usage files.

---

## 13. Compile Integration

`codectx compile` calls `SyncGlobalUsage` after the compile pipeline completes and the summary is displayed. The sync is best-effort — a failure warns to stderr and does not fail the compile.

Ordering within compile post-steps:

1. Compile pipeline runs (all stages)
2. Summary is rendered to stdout
3. Warnings are rendered (if any)
4. `SyncGlobalUsage` merges local usage into global (best-effort)

---

## 14. New Dependency

| Package | Version | Purpose |
|---------|---------|---------|
| `github.com/shirou/gopsutil/v3` | latest | Parent process name detection for caller context |

---

## 15. Design Decision Record

| Decision | Chosen | Alternative Considered | Rationale |
|----------|--------|----------------------|-----------|
| History format | Per-file JSON | JSONL append-only flat files | Per-file is atomic (no partial writes), needs no append locking, and enables filesystem-as-index via glob. JSONL would require read-parse-filter on every lookup. |
| History format | Per-file JSON | TSV flat files (previous implementation) | TSV requires escape handling, cannot represent structured fields (arrays, nested objects), and append-only files are harder to prune than individual files. |
| Filename order | `[nanoTs].[hash12].[ext]` | `[hash12].[nanoTs].[ext]` (previous) | Timestamp-first means lexicographic sort equals chronological sort. Eliminates timestamp extraction for pruning and cache lookup. |
| Cache key | `chunk_set_hash` + `compile_hash` | `chunk_set_hash` only | Compilation may change chunk content, bridges, or heading paths without changing chunk IDs. `compile_hash` invalidates stale cache entries. |
| Cache scope | `codectx generate` only | Both `query` and `generate` | Query results are fast (BM25 is in-memory after index load). Generate involves disk I/O for every chunk. Caching generate provides more value. Query caching adds complexity for minimal benefit. |
| Usage sync | On `codectx compile` only | On every invocation | Compile is an intentional, committed act. Syncing on every invocation would create noisy diffs and frequent merge conflicts in `global_usage.yml`. |
| Global usage storage | Single YAML file, checked in | Append-only log, or gitignored | Single file is simple to read and display. Merge conflicts are accepted — most projects are single-developer or have infrequent compile conflicts. |
| Caller detection | Env vars + gopsutil | Env vars only | Parent process name detection provides useful context even when tools don't set `CODECTX_CALLER`. The dependency cost is justified by the value of automatic detection. |
| File extensions | `.yml` throughout | `.yaml` | Consistency with existing codebase convention (`ai.yml`, `preferences.yml`, `hashes.yml`, `manifest.yml`). |
