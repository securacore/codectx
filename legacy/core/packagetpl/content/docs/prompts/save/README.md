# Save (AI-Assisted Commit)

This prompt stages all changes and generates a conventional commit message with an affected files summary. All commands and file references are relative to the repository root, not the directory containing this prompt file. Execute the steps in the Execution section while observing the constraints in the Rules section.

## Execution

<execution>

### 1. Stage All Changes

```bash
git add -A
```

Stage all untracked and modified files before committing.

### 2. Verify Staged Changes

```bash
git diff --staged --quiet
```

**If no staged changes exist, FAIL immediately.** Do not proceed.

### 3. Gather Context

1. `git diff --staged`: review what is being committed
2. `git log --oneline -5`: match repository commit style
3. Glob `**/WIP*.md`: if found and relevant, read for ongoing work context

### 4. Generate Commit Message

The commit message is composed of a description line followed by bracketed sections. Every line in the entire commit message must be 72 columns or fewer. This is required for referential compliance and release tooling.

**Description line:** `<type>[(scope)][!]: <description>`. Imperative mood, lowercase, no period, max 72 characters. Completes "This commit will..."

**Types:** `feat` | `fix` | `docs` | `style` | `refactor` | `perf` | `test` | `build` | `ci` | `chore`

**Scope** (optional): area affected. Example: `feat(auth):`, `fix(api):`

**Breaking changes:** Add `!` after type/scope. Example: `feat(api)!: change response format`

**`[Body]` section** (optional): explain _what_ and _why_ (not _how_), 1-4 sentences max. Separated from the description by a blank line.

**`[WIP]` section** (optional): include if any `WIP*.md` files exist and are relevant to this commit.

- Each entry starts with `- <filepath>` on its own line.
- Follow the file path with a 1-3 sentence summary on the next line, indented 2 spaces.
- Wrap continuation lines at 72 columns, also indented 2 spaces.
- Separate multiple entries with a blank line.

**`[Affected Files]` section** (required): lists every file changed in the commit as a plain unordered list using dashes. File paths only. No descriptions, annotations, or details about what changed in each file.

**Other footers** (optional):

- `BREAKING CHANGE: <explanation>`
- `Refs: #123`

### 5. Create Commit

Always write the message to a temporary file:

```bash
cat << 'EOF' > /tmp/commit-msg.txt
<type>[(scope)]: <description>

[Body]
<1-4 sentences>

[WIP]
- path/to/WIP-file.md
  <1-3 sentence summary, wrapped at 72 columns
  with 2-space indent on continuation lines>

[Affected Files]
- path/to/file-a
- path/to/file-b

<other footers>
EOF
git commit -F /tmp/commit-msg.txt
rm /tmp/commit-msg.txt
```

Omit the `[Body]` section if the description is self-explanatory. Omit the `[WIP]` section if no `WIP*.md` files are relevant.

### 6. Verify

```bash
git log -1 --oneline
```

Confirm the commit hash and message appear. If the commit failed, exit with error.

</execution>

## Rules

<rules>

- All lines in the commit message must be 72 columns or fewer. No exceptions.
- Match the repository's existing commit style
- Be specific about what changed; explain why if not obvious
- Reference related issues or tickets if known
- Always include the `[Affected Files]` section listing every file in the commit. File paths only, no descriptions or annotations.
- WIP summaries start on the line after the file path, indented 2 spaces; continuation lines also indented 2 spaces
- Do not add AI signatures, co-author lines, or emoji
- Do not pad the body or attempt to recover from failures

</rules>

## Examples

```text
fix(parser): handle ISO-8601 timestamps with timezone offset

[Body]
The regex rejected valid timestamps with +00:00 format.

[Affected Files]
- src/parser/timestamp.ts
- tests/parser/timestamp.test.ts
```

```text
feat(auth)!: require API key for all endpoints

[Body]
All endpoints now require X-API-Key header. Previously
only write operations required authentication.

[Affected Files]
- src/middleware/auth.ts
- src/routes/index.ts
- docs/API.md

BREAKING CHANGE: Unauthenticated read requests will
return 401.
```

```text
refactor(settings): extract preference validation

[Body]
Moves validation logic to shared module for reuse
across settings and profile endpoints.

[WIP]
- docs/WIP-settings.md
  Validation logic is complete. Next step is building
  the preferences UI panel and wiring it to the new
  shared module.

[Affected Files]
- src/settings/validation.ts
- src/settings/preferences.ts
- src/profile/preferences.ts
```
