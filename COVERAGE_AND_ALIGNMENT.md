# Coverage & Alignment Plan

## Phase 1: Export ShortHashLen + eliminate magic 12

- Export `ShortHashLen` from `core/history`
- Replace hardcoded `12` in `core/query/format.go` and `cmds/history/main.go`
- Add `ShortHash(hash string) string` helper to `core/history`

## Phase 2: DRY — centralize history operations

- Move `saveToHistory` workflow into `core/history` as `LogGenerate(histDir, projectDir, root, doc, chunkIDs, tokens, hash) (docPath string, err error)`
- Move `logQueryHistory` workflow into `core/history` as `LogQuery(histDir, projectDir, root, rawQuery, expandedQuery, totalResults) error`
- Both handle EnsureDir + operation + CheckAndPrune internally
- Callers (cmds/generate, cmds/query) reduce to single calls + best-effort warning
- Extract `warnHistory` to `cmds/shared/` since both generate and query use it

## Phase 3: DRY — shared resolveHistDir

- Move `resolveHistDir()` from `cmds/history/main.go` to `cmds/shared/history.go`
- Update `cmds/history`, `cmds/generate`, `cmds/query` to use shared helper

## Phase 4: Dead code cleanup

- `SaveDocument` recomputes SHA-256 that `RunGenerate` already computed — pass hash in
- Export `DocsDir` only if needed, or keep unexported (it's fine)
- Fix `parts := []string{}` to `var parts []string` in compile
- Use `strconv.Itoa` instead of `fmt.Sprintf("%d", n)` in repair

## Phase 5: Test coverage — new tests

### core/history:
- `TestHistoryDir` — basic path construction
- `TestEnsureDir` — creates dirs, idempotent
- `TestCheckAndPrune_OverLimit` — actual pruning (write >100MB, verify truncation)
- `TestAppendQuery_EmptyFields` — empty strings
- `TestAppendChunks_EmptyIDs` — empty slice
- `TestShortHash` — full hash, short hash, empty hash

### core/query:
- `TestFormatQueryResults_WithExpandedQuery` — expanded != raw
- `TestCollectSources_Duplicates` — dedup behavior
- Fix `TestRunGenerate_DeterministicHash` comment/name accuracy

### cmds/shared:
- `TestWarnHistory` — output goes to stderr
- `TestResolveHistoryDir` — returns correct path

### cmds/compile:
- `TestCountNonZero` — basic helper test

### core/project:
- No new tests needed — existing coverage is adequate for modified code

## Phase 6: Code quality fixes

- Add doc comments to unexported command vars in history
- Fix `cmds/compile/main.go` `countNonZero` doc comment
- Consistent `var x []string` vs `x := []string{}`

## Phase 7: Final verification

- `go build ./...`
- `go test ./...`
- `golangci-lint run`
- Remove this file
