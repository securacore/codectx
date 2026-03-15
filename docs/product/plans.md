# Plans

Plans are living documentation that tracks multi-step work in progress. They solve the problem of context loss — when an AI session ends mid-task, the next session shouldn't start from zero. Plans record what documentation the AI searched for, which chunks it loaded, and what progress was made at each step.

---

## How Plans Work

A plan lives in the `plans/` directory with the same structure as any other topic — a `README.md` describing the work, plus a `plan.yml` file that tracks state.

```
docs/plans/auth-migration/
  README.md          # What this plan is about
  plan.yml           # State tracking (checked into version control)
  README.spec.md     # Why these steps were chosen (optional)
```

The `plan.yml` file records:
- Plan status (draft, in-progress, blocked, completed)
- Steps with individual status, notes, and timing
- **Per-step queries** — the search terms the AI used to find relevant documentation
- **Per-step chunks** — the chunk IDs the AI loaded, directly replayable
- **Dependencies** — which documentation the plan was built against, with content hashes

### Example plan.yml

```yaml
name: "Authentication System Migration"
status: "in-progress"
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T12:00:00Z"

dependencies:
  - path: "foundation/architecture-principles"
    hash: "sha256:a1b2c3..."
  - path: "topics/authentication/jwt-tokens"
    hash: "sha256:d4e5f6..."

steps:
  - id: 1
    title: "Audit current JWT implementation"
    status: "completed"
    completed_at: "2025-03-02T14:00:00Z"
    notes: "Found 3 services using deprecated token format"
    queries:
      - "jwt token implementation current"
      - "token validation service audit"
    chunks:
      - "obj:a1b2c3.01,obj:a1b2c3.02,obj:a1b2c3.03,spec:f7g8h9.01"

  - id: 2
    title: "Implement token service refactor"
    status: "in-progress"
    started_at: "2025-03-07T09:00:00Z"
    notes: "User service and payment service updated. Order service remaining."
    queries:
      - "token service refactor implementation"
      - "order service authentication"
    chunks:
      - "obj:a1b2c3.04,obj:d4e5f6.02,obj:d4e5f6.03"
      - "obj:x9y8z7.01,spec:x9y8z7.01"

  - id: 3
    title: "Migration testing"
    status: "pending"
    blocked_by: [2]

current_step: 2
```

---

## Checking Plan Status

```bash
codectx plan status auth-migration
```

Output:

```
Plan: Authentication System Migration
Status: in-progress (step 2 of 3)
Progress: 1 step completed, 1 in progress, 1 pending

Current step: Implement token service refactor
  Started: 2025-03-07T09:00:00Z
  Notes: User service and payment service updated. Order service remaining.
  Stored queries:
    - "token service refactor implementation"
    - "order service authentication"

Dependencies:
  + foundation/architecture-principles -- unchanged
  ! topics/authentication/jwt-tokens -- content changed since last update

Blocked steps:
  Step 3 (Migration testing) -- blocked by step 2
```

Status checks are lightweight — they read `plan.yml` and compare dependency hashes without loading any documentation context.

---

## Resuming a Plan

```bash
codectx plan resume auth-migration
```

The resume flow detects whether the underlying documentation has changed since the plan was last active.

### When Documentation Is Unchanged

If all dependency hashes match, the stored chunk IDs are still valid. codectx replays the current step's chunks via `codectx generate` for instant context reconstruction:

```
Plan: Authentication System Migration
Status: in-progress (step 2 of 3)
Dependencies: all unchanged

Replaying context for step 2...
-> Generated (1,847 tokens, hash: e7f8a9b0c1d2)
  Contains: obj:a1b2c3.04, obj:d4e5f6.02, obj:d4e5f6.03, obj:x9y8z7.01, spec:x9y8z7.01

Current step: Implement token service refactor
Notes: User service and payment service updated. Order service remaining.
```

The AI has exact context reconstruction — the same chunks it was working with before, loaded instantly.

### When Documentation Has Drifted

If any dependency's content hash has changed, stored chunk IDs may be stale. codectx reports which dependencies changed and provides the stored queries:

```
Plan: Authentication System Migration
Status: in-progress (step 2 of 3)

Documentation changes since last update:
  ! topics/authentication/jwt-tokens -- content changed
  + foundation/architecture-principles -- unchanged

Stored chunks may be stale. Stored queries for current step:
  - "token service refactor implementation"
  - "order service authentication"

Recommendation: Re-run stored queries to refresh context with updated documentation.
```

The AI uses the stored queries to run `codectx query` for each one, selects new chunks from fresh results, and updates `plan.yml` with new chunks and hashes.

---

## Context Audit Trail

Even for completed steps, the stored queries and chunks create a record of *what documentation the AI relied on* to complete that work. If a problem surfaces later, the team can trace back: "Step 1 was completed using these chunks from these queries — did the AI have the right context when it made those decisions?"

This also enables plan handoffs between developers. Developer A starts the plan, their AI finds chunks through certain queries and records them. Developer B picks it up on a different machine:
- If hashes match, the exact chunks load instantly
- If docs drifted, Developer B's AI gets the stored search queries — the same search intent adapted to current documentation

---

## Design Properties

**Checked into version control.** `plan.yml` is source content, not compiled output. It's the mechanism for cross-machine, cross-developer continuity.

**Merge-friendly.** The file is declarative (current state), not accumulative (history log). History lives in git commits.

**Stable chunk IDs.** Chunk IDs are content hashes. Same source markdown + same compiler settings + same tokenizer = same chunk IDs on every machine. This is what makes chunk-level references safe to check in and replay across machines.

**Drift detection, not drift prevention.** The system detects when documentation changed under a plan and reports it. It doesn't prevent the developer from continuing — it provides information for an informed decision.

**Per-step scoping.** Resuming step 2 doesn't load step 1's chunks. Context is scoped to the current step, keeping token usage efficient.

---

[Back to overview](README.md)
