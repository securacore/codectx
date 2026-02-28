# Plans

Implementation plans for tracking multi-step work. Each plan lives in its own subdirectory under `docs/plans/` with a `README.md` entry point and a `plan.yml` state file. Plans are registered in the manifest and provide lightweight status tracking for AI triage.

## Plan Structure

Every plan has two files:

- **`README.md`**: The full plan document. Describes what needs to be done, the approach, milestones, and any open questions. This is the authoritative reference for the plan's scope and progress.
- **`plan.yml`**: A lightweight state file for quick triage. Contains the plan identifier, current status, and a brief summary. AI tools read this file to assess plan state without loading the full README.

## Directory Layout

```text
docs/plans/
  <name>/
    README.md        # Full plan document
    plan.yml         # State tracking (status, summary, dates)
```

Plan names use lowercase kebab-case. Each plan gets its own directory.

## Plan State

The `plan.yml` file tracks execution state:

```yaml
plan: <name>
status: not_started
summary: ""
started_at: ""
updated_at: ""
```

### Status Values

- **`not_started`**: Plan is defined but work has not begun.
- **`in_progress`**: Active work is underway.
- **`completed`**: All planned work is finished.
- **`blocked`**: Work cannot proceed due to an external dependency or unresolved question.

### Summary

The `summary` field is a 1-3 sentence description of the current state. AI tools use this for quick triage to determine whether to load the full plan README. Update the summary whenever the status changes or significant progress is made.

### Date Fields

- **`started_at`**: Date when work began (YYYY-MM-DD format). Set when status changes from `not_started` to `in_progress`.
- **`updated_at`**: Date of the last status or summary change (YYYY-MM-DD format). Update on every state modification.

## Lifecycle

### Create

Run `codectx new plan <name>` to scaffold a new plan. This creates the directory with a minimal `README.md` (title only) and a `plan.yml` with `status: not_started`, then updates the manifest.

After scaffolding, write the plan content in `README.md`. Describe the goal, the approach, milestones, and any dependencies or open questions.

### Read

AI tools read `plan.yml` first for triage. If the status and summary provide sufficient context, the full `README.md` does not need to be loaded. This keeps token usage low for routine status checks.

Load the full `README.md` when:

- Starting work on the plan.
- Making decisions that affect plan scope.
- Reviewing progress in detail.
- Updating the plan content.

### Update

Update the plan's state and content as work progresses:

<rules>

- Update `plan.yml` status when the work state changes. Do not leave status stale.
- Update `plan.yml` summary to reflect current state. The summary is the first thing AI reads.
- Set `started_at` when status first moves to `in_progress`. Do not change it afterward.
- Set `updated_at` on every status or summary change.
- Update `README.md` to reflect completed milestones, new decisions, and scope changes.
- Run `codectx compile` after updates to refresh the compiled documentation set.

</rules>

### Delete

Remove the plan's directory from `docs/plans/`. Run `codectx sync` or `codectx compile` to remove the stale entry from the manifest.

Only delete plans that are `completed` or no longer relevant. Completed plans serve as a historical record of past decisions and approaches.

## Manifest Registration

Plans appear in the `plans` section of `docs/manifest.yml`:

```yaml
plans:
  - id: migrate-auth
    path: plans/migrate-auth/README.md
    plan_state: plans/migrate-auth/plan.yml
    description: Migrate authentication from JWT to session-based
```

Each entry has an `id` (matches the directory name), a `path` (to the README), a `plan_state` (to the plan.yml), and a `description`. The `codectx sync` command discovers plans on disk and updates the manifest automatically.

## Best Practices

<rules>

- Keep plans focused. One plan per initiative. Do not combine unrelated work streams into a single plan.
- Write the README for a reader who has no prior context. State the goal, the approach, and the current state explicitly.
- Update the summary proactively. A stale summary misleads AI triage and wastes tokens when the full plan must be loaded unnecessarily.
- Use `blocked` status with a clear explanation. State what is blocking and what needs to happen to unblock.
- Archive completed plans by leaving them in place. Do not delete them unless they contain no useful historical context.

</rules>
