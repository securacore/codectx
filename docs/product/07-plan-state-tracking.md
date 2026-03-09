## Phase 7: Plan State Tracking

### plan.yaml Schema

```yaml
# docs/plans/auth-migration/plan.yaml
name: "Authentication System Migration"
status: "in-progress"  # draft | in-progress | blocked | completed
created: "2025-03-01T00:00:00Z"
updated: "2025-03-09T12:00:00Z"

# Documentation this plan depends on
# Each dependency tracks a content hash at the time the plan was last updated
# Used to detect documentation drift during plan resumption
dependencies:
  - path: "foundation/architecture-principles"
    hash: "sha256:a1b2c3..."
  - path: "topics/authentication/jwt-tokens"
    hash: "sha256:d4e5f6..."
  - path: "topics/authentication/oauth"
    hash: "sha256:g7h8i9..."

# Plan steps with state, context queries, and chunk references
# queries: the search terms the AI used to find relevant documentation for this step
# chunks: the codectx generate calls the AI made (each entry is one generate call)
#         directly replayable if dependency hashes haven't changed
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
    title: "Design new token schema"
    status: "completed"
    completed_at: "2025-03-05T10:00:00Z"
    queries:
      - "jwt token schema design"
      - "refresh token lifecycle"
    chunks:
      - "obj:a1b2c3.03,obj:a1b2c3.04,obj:d4e5f6.02,spec:f7g8h9.02"

  - id: 3
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

  - id: 4
    title: "Migration testing"
    status: "pending"
    blocked_by: [3]

  - id: 5
    title: "Production rollout"
    status: "pending"
    blocked_by: [4]

current_step: 3
```

**Key schema elements:**

- **dependencies with `hash`**: Each dependency records a content hash at the time the plan was last updated. This enables drift detection — the CLI can tell whether the documentation the plan was built against has changed.

- **Per-step `queries`**: The search terms the AI used to find relevant documentation for each step. These are preserved so that if the plan resumes and hashes have changed, the AI has the original search intent to re-run against the updated documentation rather than searching blind.

- **Per-step `chunks`**: Each entry is a comma-delimited string of chunk IDs matching the `codectx generate` input format — directly replayable. Multiple entries mean the AI made multiple generate calls during that step. If dependency hashes haven't changed, these chunk IDs are still valid and can be replayed for instant context reconstruction.

- **Pending steps have no queries or chunks**: These are populated by the AI as it begins working on each step.

### Resumption Flow

`codectx plan resume auth-migration` performs the following:

1. Read plan.yaml, identify current_step (step 3)
2. Check each dependency's current content hash against its `hash`
3. **If all hashes match** (documentation unchanged):
   - Replay step 3's chunks via `codectx generate` for each entry
   - Return assembled context plus plan state
   - AI has exact context reconstruction — instant resumption
4. **If any hashes changed** (documentation drifted):
   - Report which dependencies changed:
     ```
     Plan: Authentication System Migration
     Status: in-progress (step 3 of 5)
     
     Documentation changes since last update:
       ⚠ topics/authentication/jwt-tokens — content changed
       ✓ foundation/architecture-principles — unchanged
       ✓ topics/authentication/oauth — unchanged
     
     Stored queries for current step:
       - "token service refactor implementation"
       - "order service authentication"
     
     Recommendation: Review changes to jwt-tokens before continuing.
     Re-run stored queries to refresh context with updated documentation.
     ```
   - AI uses the stored queries to run `codectx query` for each, selects new chunks from fresh results
   - AI updates plan.yaml with new chunks and hashes once it has re-established context

### Context Audit Trail

Even for completed steps, the stored queries and chunks create a valuable record of *what documentation the AI relied on* to complete that work. If a problem surfaces later, the team can trace back: "Step 2 was completed using these chunks from these queries — did the AI have the right context when it made those decisions?"

This also enables plan handoffs between developers. Developer A starts the plan, their AI finds chunks through certain queries and logs them. Developer B picks it up on a different machine. If hashes match, the exact chunks load. If not, developer B's AI doesn't need to figure out *what to search for* from scratch — the stored queries give it the same search intent, adapted to the current documentation state.

### Design Considerations

- **plan.yaml must be merge-friendly**: Keep it declarative (current state), not accumulative (history log). History lives in git commits.
- **plan.yaml is checked into version control**: This is source content, not compiled output. It's the mechanism for cross-machine, cross-developer continuity.
- **Token cost**: plan.yaml is small enough to load directly. No chunking needed for plan state files.
- **Chunk IDs are stable across machines**: Chunk IDs are content hashes of the chunk text. Same source markdown + same preferences.yaml + same tokenizer encoding = same chunks, same hashes, on every machine. This is what makes chunk-level references safe to check in.
- **Drift detection, not drift prevention**: The system detects when documentation changed under a plan and reports it. It doesn't prevent the developer from continuing — it provides the information for an informed decision.

---

