# Coverage and Alignment Plan

## Tasks

### 1. Dead Parameter Removal
- [ ] Remove unused `ordered bool` parameter from `renderListItemContent` in parse.go
- [ ] Update callers of `renderListItemContent` to stop passing the parameter
- [ ] Remove unused `*scaffold.Result` parameter from `buildSummaryTree` in cmds/init/main.go

### 2. Dead Code Removal
- [ ] Remove `tui.NotDetectedTool` (only used in its own test)
- [ ] Remove `tui.IconBullet` (only consumer was NotDetectedTool)
- [ ] Update tui tests to remove references to removed items

### 3. DRY: Consolidate ChunkType Metadata
- [ ] Replace duplicate switch statements in idPrefix() and OutputDir() with shared lookup table

### 4. DRY: Centralize Default Root Resolution
- [ ] Add project.ResolveRoot(root string) string helper
- [ ] Replace 6 occurrences of the if root == "" pattern

### 5. Magic Strings -> Constants
- [ ] detect.go: Use DefaultModel constant where model string is inline in knownProviders
- [ ] detect.go: Use DefaultEncoding in EncodingForModel default case
- [ ] config.go: Add DefaultVersion = "0.1.0", DefaultOutputFormat = "markdown"

### 6. Nil Safety
- [ ] chunk.RenderMeta / Render: handle nil chunk
- [ ] chunk.OutputFilename / OutputPath: handle nil chunk  
- [ ] markdown.Strip: handle nil document
- [ ] Note: ValidateFile and RootDir are dead code, skip nil guards

### 7. Error Consistency
- [ ] Standardize "document is nil" errors to use errors.New (no wrapping needed)

### 8. Missing Test Coverage
- [ ] Test chunk.OutputFilename with empty ID
- [ ] Test chunk.RenderMeta with nil chunk (after adding nil guard)

### 9. Run Linter + Final Validation
- [ ] golangci-lint run ./...
- [ ] go test -race ./... -count=1
- [ ] go vet ./...
- [ ] Verify test counts and coverage
