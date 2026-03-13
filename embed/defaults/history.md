# History

codectx logs every query and generate invocation as structured JSON entries. Generated documents are cached — repeated `codectx generate` calls with the same chunk IDs against the same compilation state are served from cache.

## Commands

```
codectx history
```

Shows recent query and generate activity (last 10 of each), including caller, session, and compile staleness.

```
codectx history show <hash>
```

Prints a previously generated document to stdout. The hash is a prefix match against the document's content hash shown in generate output.

```
codectx history queries
codectx history chunks
codectx history clear
```

Filtered views and cleanup. `clear` removes all history data (does not affect usage metrics).

```
codectx usage
```

Shows local machine and project lifetime token usage metrics, including invocation counts, cache hit rates, and usage by caller and model.

## When to Use

If the user references earlier context, a previous search, or a document they fetched before, check history first. The hash from `codectx generate` output identifies each document uniquely. Use `codectx usage` to understand token consumption patterns.
