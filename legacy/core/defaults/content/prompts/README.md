# Prompts

AI-executable prompt definitions for automated tasks. Each prompt lives in its own subdirectory under `docs/prompts/` with a `README.md` entry point. Prompts are registered in the manifest and can be invoked through the project's task runner or AI tool CLI.

For the authoring conventions that govern prompt structure, see [ai-authoring](../ai-authoring/README.md) (Prompt-Specific Patterns section).

## Prompt Structure

Every prompt follows a consistent format:

- **Purpose statement** (1-2 sentences): what the prompt does and when to use it.
- **Execution section** wrapped in `<execution>` tags: sequential steps the AI carries out.
- **Rules section** wrapped in `<rules>` tags: constraints the AI observes during execution.
- **Examples** (optional): concrete input/output samples for complex prompts.

Each step in the execution section is numbered and has a clear completion criterion. The final step is always a verification step that confirms the output is correct.

## Directory Layout

```text
docs/prompts/
  <name>/
    README.md        # Prompt definition (execution + rules)
```

Prompt names use lowercase kebab-case. Each prompt gets its own directory to allow future expansion (supporting files, templates, etc.) without restructuring.

## Lifecycle

### Create

Run `codectx new prompt <name>` to scaffold a new prompt. This creates the directory and a minimal `README.md` with a title, then updates the manifest.

After scaffolding, write the prompt content following the structure described above. The `<execution>` section contains the steps; the `<rules>` section contains the constraints. See the existing `save` prompt for a complete reference implementation.

### Invoke

Prompts are invoked through the project's task runner:

- **Claude Code:** `just claude prompt <name> "<context>"`
- **OpenCode:** `just opencode prompt <name> "<context>"`
- **AI abstraction:** `just ai save` (delegates to the configured AI tool)

The task runner reads the prompt's `README.md` and passes it to the AI tool along with the provided context string. The AI tool executes the prompt's steps while observing the constraints.

### Update

Edit the prompt's `README.md` directly. Run `codectx compile` afterward to update the compiled documentation set. The manifest is updated automatically during compilation via sync.

When updating a prompt:

<rules>

- Preserve the `<execution>` and `<rules>` tag structure. Do not merge execution steps and constraints into a single section.
- Keep steps numbered sequentially. Do not skip numbers.
- Update verification steps to match any changes to earlier steps.
- Run the prompt once after editing to verify it executes correctly.

</rules>

### Delete

Remove the prompt's directory from `docs/prompts/`. Run `codectx sync` or `codectx compile` to remove the stale entry from the manifest.

## Manifest Registration

Prompts appear in the `prompts` section of `docs/manifest.yml`:

```yaml
prompts:
  - id: save
    path: prompts/save/README.md
    description: Stage all changes and generate a conventional commit
```

Each entry has an `id` (matches the directory name), a `path` (relative to the docs root), and a `description` (one-line purpose statement). The `codectx sync` command discovers prompts on disk and updates the manifest automatically.

## Authoring Guidelines

<rules>

- Open with a clear purpose statement. One to two sentences that explain what the prompt does and when to use it.
- Separate execution from constraints. Use `<execution>` tags for steps and `<rules>` tags for constraints. Do not mix them.
- Specify what to omit. Models tend to over-include. Name the things the model must not do.
- Gate sequential dependencies. State "Do not proceed to Step N until Step N-1 is complete" when order matters.
- End with a verification step. The last execution step confirms the output is correct.
- All commands and file references are relative to the repository root, not the prompt file location.
- Do not add AI signatures, co-author lines, or emoji in prompt output.

</rules>
