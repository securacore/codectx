# CLI Usage

## Search

```
codectx query "your search terms"
```

Returns ranked results from three categories: instruction chunks (how-to), reasoning chunks (why), and system chunks (compiler behavior). Each result shows a chunk ID, heading, token count, and relevance score.

## Retrieve

```
codectx generate "obj:id1,obj:id2,spec:id3"
```

Assembles requested chunks into a single document printed to stdout. Summary (token count, content hash, history path) goes to stderr. Use chunk IDs from query results. The output includes heading hierarchy, bridge summaries at content gaps, and a list of related chunks not included.

Use `--file <path>` to write the document to a file instead of stdout (summary goes to stdout in this mode).

## History

```
codectx history
```

Shows recent query and generate activity. Use `codectx history show <hash>` to retrieve a previously generated document by its content hash prefix.

## Repair

```
codectx repair
```

Restores missing directories and default system files. Run after accidental deletions or codectx upgrades.

## When to Search

Search before writing code for any task involving project-specific patterns, conventions, or architecture. If the query returns relevant results, retrieve and read them before proceeding. If the user references earlier context, check `codectx history` first.
