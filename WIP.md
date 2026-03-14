# WIP: `codectx prompt` Command

> Auto-query + auto-generate in a single command.
> Eliminates the failure mode where AI runs `codectx query` but skips `codectx generate`.

## Problem

AI agents run `codectx query`, see chunk headings, judge them irrelevant by title alone, and
skip `codectx generate` entirely. The documentation protocol says "MUST generate before judging
relevance" but the AI's own optimization instinct overrides this. A single command that performs
both operations atomically makes correct behavior the path of least resistance.

## Command

```
codectx prompt "How do I create a React component"
codectx prompt --delta 0.2 "How do I create a React component"
codectx prompt --delta -0.1 --budget 2000 "architecture patterns"
codectx prompt --file /tmp/context.md "error handling"
```

## CLI Flags

| Flag         | Type      | Description                                                            |
|--------------|-----------|------------------------------------------------------------------------|
| `--delta`    | `float64` | One-off override of `budget_delta` from preferences (replaces, not additive) |
| `--budget`   | `int`     | Hard override — bypasses the formula entirely                          |
| `--file`     | `string`  | Write document to file instead of stdout                               |
| `--top`      | `int`     | Max query results to consider                                          |
| `--no-cache` | `bool`    | Skip generate cache lookup                                             |

### Flag Precedence

```
if --budget flag set:
    budget = flag value                          # hard override
else:
    delta = --delta flag if set, else config.BudgetDelta
    budget = chunk_target × config.BudgetMultiplier × (1 + delta)
```

`--budget` is the escape hatch that bypasses the formula entirely. `--delta` is the
incremental adjustment for the formula. They don't interact — `--budget` wins if both are set.

## Budget Calculation

### Formula

```
budget = chunk_target × budget_multiplier × (1 + budget_delta)
```

Where:
- `chunk_target` = `preferences.chunking.target_tokens` (default 450)
- `budget_multiplier` = configurable (default 3)
- `budget_delta` = configurable scaling factor (default 0.0)

### Examples (chunk_target=450, multiplier=3)

| Delta | Calculation    | Budget | ~Chunks |
|-------|----------------|--------|---------|
| -0.2  | 1350 × 0.8     | 1,080  | ~2      |
|  0.0  | 1350 × 1.0     | 1,350  | ~3      |
|  0.1  | 1350 × 1.1     | 1,485  | ~3      |
|  0.2  | 1350 × 1.2     | 1,620  | ~3-4    |
|  0.5  | 1350 × 1.5     | 2,025  | ~4-5    |
|  1.0  | 1350 × 2.0     | 2,700  | ~6      |

### Design Rationale

The budget is derived from the project's configured chunk size rather than a fixed number.
This makes it deterministic and adaptive — projects with larger chunks get proportionally
larger budgets, always targeting roughly the same number of chunks. The multiplier of 3 is
the baseline because BM25 relevance drops off quickly after the top few results, and 3 chunks
is sufficient for most focused queries.

## Auto-Selection Algorithm

```go
func selectChunks(results []ResultEntry, budget int) []string {
    var selected []string
    total := 0
    for _, r := range results {
        if len(selected) > 0 && total+r.Tokens > budget {
            break
        }
        selected = append(selected, r.ChunkID)
        total += r.Tokens
    }
    return selected
}
```

Key behavior: the first result is always included even if it exceeds the budget. After that,
chunks are added greedily (by score order) until the budget is exceeded.

## Config

### `preferences.yml`

```yaml
prompt:
  budget_multiplier: 3      # base chunk count target
  budget_delta: 0.0          # default scaling factor: 0.1 = +10%, -0.2 = -20%
```

### `PromptConfig` struct (`core/project/config.go`)

```go
// PromptConfig controls the prompt command's auto-selection behavior.
type PromptConfig struct {
    // BudgetMultiplier is the base number of chunks to target.
    // Budget = chunk_target_tokens × BudgetMultiplier × (1 + BudgetDelta).
    // Defaults to 3.
    BudgetMultiplier float64 `yaml:"budget_multiplier"`

    // BudgetDelta scales the budget up or down incrementally.
    // 0.0 = no change, 0.1 = +10%, -0.2 = -20%.
    // Defaults to 0.0.
    BudgetDelta float64 `yaml:"budget_delta"`
}

// EffectiveBudget computes the token budget for chunk auto-selection.
// deltaOverride, if non-nil, replaces the configured BudgetDelta (for --delta flag).
func (p PromptConfig) EffectiveBudget(chunkTarget int, deltaOverride *float64) int {
    mult := p.BudgetMultiplier
    if mult <= 0 {
        mult = 3
    }
    delta := p.BudgetDelta
    if deltaOverride != nil {
        delta = *deltaOverride
    }
    budget := float64(chunkTarget) * mult * (1 + delta)
    if budget < float64(chunkTarget) {
        return chunkTarget
    }
    return int(math.Ceil(budget))
}
```

## Output Format

```
-> Prompt: "How do I create a React component"
   Expanded: react compon ui widget
   Selected: 3 chunks (1,247 tokens) from 8 query results
   Budget: 1,350 tokens (450 × 3 × 1.0)

--- document content (full markdown) ---

---
Generated (1,247 tokens, hash: a1b2c3d4e5f6)
  History: docs/.codectx/history/docs/...
  Contains: obj:abc123.03, obj:abc123.04, spec:f7g8h9.02

  Related chunks not included:
    obj:abc123.02 — Auth > JWT > Token Structure (488 tokens)
```

Scored results only are auto-generated. Related/adjacent chunks are shown as follow-up
suggestions in the footer, not auto-included.

## Command Flow

```
1.  Parse args (search terms)
2.  Discover project, load config + preferences
3.  Resolve compiled dir, encoding, history dir, caller
4.  Compute budget:
    a. chunkTarget = preferences.Chunking.TargetTokens (or 450)
    b. deltaOverride = --delta flag (nil if unset)
    c. If --budget flag set: budget = flag value
       Else: budget = preferences.Prompt.EffectiveBudget(chunkTarget, deltaOverride)
5.  Run query (unified or flat based on indexer config)
6.  Log query to history
7.  If zero results → "No results found", return
8.  Auto-select: iterate results by score, accumulate until budget exceeded
    (always include first result)
9.  Check generate cache for selected chunk IDs
10. Run generate with selected chunk IDs
11. Log generate to history
12. Output: summary header → document content → related chunks
13. Update usage (query + generate)
```

## History & Usage

Both a query log entry AND a generate log entry are written, same as if the AI ran both
commands manually. This preserves the full audit trail.

- `history.LogQuery(...)` — raw/expanded query, result count, compile hash, caller
- `history.LogGenerate(...)` — document bytes, chunk IDs, tokens, content hash, caller
- `usage.UpdateQuery(...)` + `usage.UpdateGenerate(...)` — invocation counts
- Generate cache works normally via `history.GenerateCacheLookup(...)`

## Files to Create

| File                  | Description     |
|-----------------------|-----------------|
| `cmds/prompt/main.go` | Command handler |

## Files to Modify

| File                                          | Change                                                      |
|-----------------------------------------------|-------------------------------------------------------------|
| `main.go`                                     | Import + register `promptcmd.Command`                       |
| `core/project/config.go`                      | Add `PromptConfig` struct, add to `PreferencesConfig`, defaults, `EffectiveBudget()` |
| `core/query/format.go`                        | Add `FormatPromptSummary()` for combined output             |
| `embed/defaults/documentation-protocol.md`    | Add `codectx prompt` as primary quick-search path           |
| `core/query/format_test.go`                   | Tests for `FormatPromptSummary()`                           |
| `core/project/config_test.go`                 | Tests for `PromptConfig.EffectiveBudget()`                  |

## Documentation Protocol Update

Restructure the workflow in `documentation-protocol.md`:

- **Step 1: Prompt (quick search)** — `codectx prompt "your search terms"` — recommended default
- **Step 2: Query + Generate (detailed search)** — for fine-grained control when prompt results are insufficient
- **Step 3: Act** — unchanged
- **Step 4: Re-query** — updated to mention both `codectx prompt` and `codectx query`
- **Step 5: Validate** — unchanged

The existing query+generate steps remain as the "detailed" alternative path.

## Edge Cases

- **Zero query results** — output "No results found" same as `codectx query`
- **All results exceed budget** — always include at least the top 1 result
- **`--budget` and `--delta` both set** — `--budget` wins, `--delta` is ignored
- **Preferences not loadable** — fall back to `PromptConfig{}` zero value, `EffectiveBudget()` defaults multiplier to 3
- **`budget_multiplier` set to 0 or negative** — defaults to 3
- **Computed budget less than one chunk** — floor at `chunkTarget` (always at least 1 chunk's worth)

## Relationship to Existing Commands

`codectx prompt` does not replace `codectx query` or `codectx generate`. Those remain for
when the AI needs fine-grained control (selectively generating specific chunks, exploring
query results before deciding, generating chunks from multiple queries into one document).
`codectx prompt` is the default "quick search" path — one command, full pipeline.
