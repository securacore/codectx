# History

codectx logs every query and generate invocation. Use this when the user asks about previous searches, wants to recall earlier documentation, or needs to re-examine context from a prior task.

## Commands

```
codectx history
```

Shows recent query and generate activity (last 10 of each).

```
codectx history show <hash>
```

Prints a previously generated document to stdout. The hash is a prefix match against the document's content hash shown in generate output.

```
codectx history queries
codectx history chunks
codectx history clear
```

Filtered views and cleanup. `clear` removes all history data.

## When to Use

If the user references earlier context, a previous search, or a document they fetched before, check history first. The hash from `codectx generate` output identifies each document uniquely.
