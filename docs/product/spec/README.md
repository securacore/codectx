# Product Architecture Specification

Spec for the product architecture documentation. For the architecture itself, see [README.md](../README.md).

## Purpose

codectx is a documentation package manager that does not yet exist. Before building it, every architectural decision must be recorded so AI agents and engineers can implement the system consistently across sessions. Without this spec, each session would make independent decisions about configuration format, package structure, resolution strategy, compilation semantics, and CLI design.

## Decisions

- **`codectx.yml` as sole configuration, YAML format.** A single file at the repository root is the source of truth for dependencies, activation state, and build settings. YAML is consistent with the project's documentation metadata files. A single config eliminates sync issues between multiple files. Alternative considered: separate `codectx.json` for dependencies and `docs.yml` for documentation mapping (rejected; two files that must stay synchronized create a maintenance burden and a category of bugs).

- **`package.yml` as data map.** Every package (local, installed, compiled) has a `package.yml` that indexes its documentation entries. This is the navigation layer that makes AI consumption token-efficient: AI reads the small data map to understand what exists, then loads specific documents on demand. Alternative considered: no manifest, let AI scan directory trees (rejected; scanning wastes tokens and provides no dependency or loading metadata). Alternative considered: JSON format (rejected; YAML is more readable for the documentation-heavy content and consistent with the project).

- **`.codectx/` for compiled output.** The compiled documentation set lives in a hidden directory at the repository root, separate from the source in `docs/`. Separation ensures the source (editable, with packages/) and artifact (generated, self-contained) never mix. The dot prefix keeps it unobtrusive in directory listings. Alternative considered: compiling into `docs/` itself (rejected; mixes source and artifact, makes it unclear which files are generated). Alternative considered: `build/` or `dist/` (rejected; these conventionally hold code build artifacts, not documentation).

- **Git-first package resolution, registry-ready design.** Packages are fetched from Git repositories. The naming convention (`name@author`) and configuration format are designed so a package registry can be added later without breaking changes. Git-first means no infrastructure is needed to ship a working tool. Alternative considered: registry from the start (rejected; premature infrastructure that delays shipping). Alternative considered: Git-only with no registry path (rejected; forecloses a valuable future option without good reason).

- **Semver versioning.** Industry standard for version constraints. Compatible with Git tags (v1.0.0), range syntax (^1.0.0, ~1.0.0), and future registry resolution. Alternative considered: Git commit hashes only (rejected; no semantic meaning, cannot express compatibility ranges). Alternative considered: date-based versioning (rejected; no compatibility semantics).

- **`name@author:version` naming convention.** The at-sign separates the author namespace from the package name. The colon separates the version. This avoids ambiguity in parsing: `@` never appears in package names or versions, `:` never appears in package names or author names. Alternative considered: npm-style `@author/name@version` (rejected; uses `@` for two different purposes, parsing ambiguity). Alternative considered: Go-style `github.com/author/name` (rejected; too verbose for a documentation-focused tool, tied to a single hosting provider).

- **Project documentation IS a local package.** The project's own documentation in `docs/` follows the same `package.yml` format as installed packages. One format for everything means no special cases in the compile step, and any project's documentation can be extracted and published as a package. Alternative considered: separate format for local docs (rejected; creates two code paths in the compiler and prevents publishing local docs as a package).

- **Activation-based conflict handling, not file-level.** Packages are namespaced (`name@author/`), so file-level conflicts are impossible. Conflicts only occur at the activation level when two packages provide documentation for the same domain. The CLI detects overlaps during `codectx add` and prompts the user to resolve them interactively. Alternative considered: last-write-wins priority ordering (rejected; implicit behavior that leads to silent overrides). Alternative considered: error-and-abort on any overlap (rejected; too strict, many overlaps are intentional and resolvable).

- **Interactive activation during `codectx add`.** When a package is added, the CLI reads its `package.yml`, presents its contents, and prompts the user to choose what to activate. The user can activate all, select specific entries, or activate none. This ensures activation is always explicit and the user understands what they are adding to their compiled output. Alternative considered: activate everything by default (rejected; risks unintended conflicts and bloated compiled output). Alternative considered: activate nothing by default, require manual config editing (rejected; poor developer experience).

- **`plans/` with `state.yml` for status tracking.** Plans are implementation documents (feature plans, system blueprints) that can be large. The `state.yml` file provides a lightweight status summary that AI can read without loading the full plan. This enables efficient triage: "which plans are in progress?" costs a few hundred tokens instead of loading every plan document. Alternative considered: status in `package.yml` (rejected; mixes navigation metadata with mutable state). Alternative considered: no state tracking (rejected; AI has no way to assess plan relevance without loading every plan).

- **`codectx link` as a separate command.** Creating AI tool entry point files (CLAUDE.md, AGENTS.md) is a distinct operation from compiling documentation. It involves backing up existing files and creating new ones with a specific format. Separating it from compile means the user controls when entry points are updated and can run compile without modifying entry point files. Alternative considered: part of `codectx compile` (rejected; compile should be safe to run repeatedly without side effects on files outside its output directory). Alternative considered: part of `codectx init` (rejected; init runs once, but entry points may need updating when the output directory changes).

- **Compiled output uses a distinct format.** The `.codectx/` directory uses a compiled format distinct from the source package format. It writes `manifest.yml` (not `package.yml`) with content-addressed object references, provenance tracking, and optional sub-manifest decomposition. The compiled manifest is validated by `compiled.schema.json`. A `heuristics.yml` sidecar provides size and token estimates for tooling. This means the compiled output is self-describing and optimized for AI consumption. Alternative considered: reusing the source package format for compiled output (rejected; the compiled format needs content-addressed paths, provenance, and decomposition support that the source format does not have). Alternative considered: flat file dump (rejected; loses the data map, dependencies, and loading metadata).

- **`codectx.lock` from compilation.** The lock file records the exact resolved versions, checksums, and activation state so the compiled output can be reproduced deterministically. `codectx add --lockfile` reads the lock file to reproduce the exact package state. Alternative considered: no lock file, rely on version pinning in codectx.yml (rejected; version ranges mean different resolutions at different times). Alternative considered: lock file from install, not compile (rejected; the compile step is when the full resolved state is known).

- **Minimal initial CLI (init, add, compile, link, version).** Ship the core workflow first: create config, add packages, compile output, link to AI tools, show version. Commands like remove, list, update, and search are deferred until usage patterns reveal what is actually needed. Alternative considered: full command set from the start (rejected; speculative commands waste development time and may not match real usage patterns).

- **`Name()`/`SetName()` getter/setter convention, following Go standards.** Although this is a Go code decision documented in the Go topic, it reflects the broader principle applied to this product: follow established conventions of the ecosystem rather than inventing alternatives. This applies equally to the package format (YAML like other documentation metadata), versioning (semver like every package manager), and naming (namespaced identifiers like modern package managers).

## Dependencies

- [docs/foundation/philosophy.md](../../foundation/philosophy.md): guiding principles ("Configuration is Truth," "Leverage Before Building," "Abstractions Must Earn Their Place")
- [docs/foundation/specs.md](../../foundation/specs.md): spec template this document follows
- [docs/schemas/codectx.schema.json](../../schemas/codectx.schema.json): formal definition of codectx.yml
- [docs/schemas/package.schema.json](../../schemas/package.schema.json): formal definition of package.yml
- [docs/schemas/state.schema.json](../../schemas/state.schema.json): formal definition of state.yml

## Structure

- `README.md`: product architecture overview (the system design)
- `spec/README.md`: this file; reasoning and decisions behind the architecture
