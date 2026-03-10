# Coverage and Alignment Plan

## Before

- **Tests:** 163 pass, 0 fail
- **Linter:** 0 issues
- **Overall coverage:** 74.1%
- **Package coverage:**
  - `core/tui`: 100%
  - `embed`: 100%
  - `core/markdown`: 93%
  - `core/detect`: 87%
  - `core/project`: 87%
  - `core/scaffold`: 87.1%
  - `cmds/init`: 12.9% (only unexported helpers testable; `run()` is TUI-bound)
  - `cmds/version`: 0% (3-line command, untestable via unit tests)
  - `main`: 0% (entry point)

## After

- **Tests:** 175 pass, 0 fail
- **Linter:** 0 issues
- **Build:** clean
- **Overall coverage:** 75.3%
- **Package coverage:**
  - `core/tui`: 100% (maintained)
  - `embed`: 100% (maintained)
  - `core/markdown`: 95.9% (was 93%)
  - `core/detect`: 87.7% (was 87%)
  - `core/project`: 94.4% (was 87%)
  - `core/scaffold`: 87.1% (maintained)
  - `cmds/init`: 11.1% (was 12.9% -- `encodingForModel` tests moved to `core/detect` where the function now lives)
  - `cmds/version`: 0% (3-line command, untestable via unit tests)
  - `main`: 0% (entry point)

## Actions Taken

### 1. DRY: Extract shared `writeYAMLFile` helper in `core/project/config.go`

Three `WriteToFile` methods followed identical marshal-prepend-mkdir-write patterns. Extracted to single `writeYAMLFile()` helper. Fixed inconsistency where `Config.WriteToFile` did not create parent directories but the other two did -- all three now consistently ensure parent directories exist.

### 2. DRY: Centralize model/encoding registry in `core/detect`

- Exported `DefaultModel` and `DefaultEncoding` constants from `core/detect`
- Exported `EncodingForModel()` function from `core/detect`
- Removed duplicate `encodingForModel()` from `cmds/init`
- Updated `core/project/config.go` to use `detect.DefaultModel` and `detect.DefaultEncoding` instead of hardcoding the model string
- Updated `recommendModel()` in detect to use `DefaultModel` const instead of hardcoded strings
- Moved encoding tests from `cmds/init` to `core/detect` where the function now lives

### 3. Remove no-op function `stripEmphasisMarkers` in `core/markdown/strip.go`

Removed the 13-line no-op function and its call site. Added a comment at the call site explaining why no stripping is needed (goldmark AST already handles it).

### 4. Remove `nodeText` trivial wrapper in `core/markdown/parse.go`

Removed the 3-line wrapper function that only called `renderInlineText`. Updated the two call sites in `block.go` to call `renderInlineText` directly.

### 5. Unexport unused TUI theme symbols -- CANCELLED

Assessed and decided against. `IconArrow`, `StyleSuccess`, `StyleInfo`, `StyleAccent`, `StyleHeading`, and `NotDetectedTool` are part of the intentional shared theme API surface. They exist for future commands to use and unexporting them would create unnecessary churn when those commands are added. These are palette elements, not dead code.

### 6. Improved test coverage

- Added `TestParse_BlockquoteWithCodeBlock` and `TestParse_BlockquoteWithList` -- exercises the `default` branch in `renderBlockquoteText` (was 50%, now covered)
- Added `TestConfig_WriteToFile_CreatesParentDirs` -- verifies the new consistent `MkdirAll` behavior
- Added `TestConfig_WriteToFile_ErrorOnInvalidPath` -- tests the error path
- Added `TestResult_HasTools` and `TestResult_HasProviders` -- explicit tests for these helpers (previously only tested implicitly via `HasAnything`)
- Added `TestDefaultModel_IsSet` and `TestDefaultEncoding_IsSet` -- validates the exported constants
- Added 7 `TestEncodingForModel_*` tests to `core/detect` (moved from `cmds/init`)

### 7. Linter: 0 issues (clean before and after)

### 8. Final verification: all tests pass, linter clean, build succeeds
