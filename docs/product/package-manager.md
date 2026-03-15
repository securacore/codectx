# Package Manager

codectx includes a package manager for installing, sharing, and managing documentation. Packages are pure markdown content hosted on GitHub. When you install a package, your local compiler processes it alongside your own documentation — one set of compilation rules for everything.

---

## What a Package Is

A package is curated documentation content — nothing more:

```
my-package/
  codectx.yml          # Name, org, version, description, dependencies
  foundation/          # Optional
  topics/              # Optional
  plans/               # Optional
  prompts/             # Optional
```

No `system/` directory. No `.codectx/` directory. No compiler configuration. No AI instructions. Packages are pure content that gets processed by your compiler with your settings.

Package authoring has near-zero friction. Write markdown. Organize it into the standard directories. Publish. You don't need to understand compilation, BM25, or taxonomy extraction. The quality of the compiled output depends on the structure and content of the markdown itself.

---

## The GitHub Registry

Packages are GitHub repositories using the `codectx-[name]` naming convention. No custom registry infrastructure — GitHub provides hosting, versioning (git tags), discovery (API search), and built-in quality signals (stars, issues, activity).

**Naming convention**: A package named `react-patterns` under org `community` lives at `github.com/community/codectx-react-patterns`. The dependency reference `react-patterns@community` maps to this repo automatically.

**Versioning**: Git tags are versions. Tag `v2.3.1` on the repo corresponds to version `2.3.1` in `codectx.yml`. The `latest` reference resolves to the most recent semver tag.

---

## Finding Packages

Search for packages on GitHub:

```bash
codectx search "react patterns"
```

Output:

```
Search results for: "react patterns"

1. react-patterns@community (v2.4.0)
   github.com/community/codectx-react-patterns
   React component and hook patterns for AI-driven development

2. react-testing@community (v1.1.0)
   github.com/community/codectx-react-testing
   Testing patterns and strategies for React applications

3. react-nextjs@webteam (v3.0.2)
   github.com/webteam/codectx-react-nextjs
   Next.js and React integration patterns
```

---

## Adding and Installing Packages

Add a dependency to your project:

```bash
codectx add react-patterns@community
```

This updates `codectx.yml` with the dependency. Install all declared dependencies:

```bash
codectx install
```

The installer:
1. Resolves `[name]@[org]` to `github.com/[org]/codectx-[name]`
2. Resolves version tags (`latest` becomes the highest semver tag)
3. Downloads to `.codectx/packages/`
4. Generates or updates `codectx.lock` with resolved versions and commit SHAs
5. Resolves transitive dependencies

After installing, run `codectx compile` to process the new content alongside your local docs.

### Removing Packages

```bash
codectx remove react-patterns@community
```

---

## Dependencies in codectx.yml

```yaml
# Project codectx.yml
dependencies:
  react-patterns@community:latest:
    active: true
  company-standards@acme:2.0.0:
    active: true
  tailwind-guide@designteam:2.1.0:
    active: true
  legacy-api-docs@internal:1.0.0:
    active: false    # Installed but excluded from compilation
```

### Active/Inactive Toggle

Packages can be toggled active or inactive. Inactive packages remain installed in `.codectx/packages/` but are excluded from compilation — their chunks aren't indexed, their taxonomy terms aren't included, and they consume zero tokens at runtime.

This lets you install many reference packages and selectively enable only what's relevant to your current work. When a direct dependency is deactivated, its transitive-only dependencies also deactivate. If a transitive dependency is also required by another active direct dependency, it stays active.

---

## Updating Packages

Re-resolve all dependencies to their latest compatible versions:

```bash
codectx update
```

Output:

```
Resolving dependencies...
  react-patterns@community: 2.3.1 -> 2.4.0 (updated)
    -> github.com/community/codectx-react-patterns@v2.4.0
  company-standards@acme: 2.0.0 (unchanged)
  tailwind-guide@designteam: 2.1.0 (unchanged)
  javascript-fundamentals@community: 1.3.0 (transitive, unchanged)

Updated codectx.lock
Downloaded: react-patterns@community:2.4.0

Recompiling (1 package changed)...
Compiled: 348 files -> 4,803 chunks (2,178,200 tokens)
```

`codectx update` handles the full cycle: resolve, download, and recompile if content changed.

---

## The Lock File

`codectx.lock` captures the full flattened dependency tree with exact resolved versions. It is checked into version control for deterministic reproducibility.

```yaml
# codectx.lock (auto-generated)
lockfile_version: 1
resolved_at: "2025-03-09T12:00:00Z"

packages:
  react-patterns@community:
    resolved_version: "2.3.1"
    repo: "github.com/community/codectx-react-patterns"
    commit: "a1b2c3d4e5f6..."
    source: "direct"

  javascript-fundamentals@community:
    resolved_version: "1.3.0"
    repo: "github.com/community/codectx-javascript-fundamentals"
    commit: "g7h8i9j0k1l2..."
    source: "transitive"
    required_by:
      - "react-patterns@community:2.3.1"
```

The `source` field distinguishes direct dependencies from transitive ones. The `required_by` field traces the dependency chain. The git commit SHA pins exact content.

**Install behavior**: If `codectx.lock` exists and `codectx.yml` hasn't changed, `codectx install` uses the lock file (fast, deterministic). If `codectx.yml` changed, it re-resolves affected entries and updates the lock.

---

## Transitive Dependencies

Published packages can declare their own dependencies. When you install a package, its dependencies are resolved and installed transitively.

All dependencies — direct and transitive — are flattened into `.codectx/packages/` at the same level. No nesting. If two packages depend on the same package at compatible semver ranges, the resolver picks the highest compatible version and installs it once. If two packages have incompatible version requirements, the installer warns and you resolve the conflict manually.

Flat resolution is safe for documentation packages. Documentation content is additive, not behaviorally breaking. A fundamentals package at version 1.2.0 vs 1.3.0 probably added new topics but didn't break existing ones.

---

## Publishing

Publish your package to GitHub:

```bash
codectx publish
```

This reads `codectx.yml` for name, org, and version; validates the directory structure; creates a git tag for the version; and pushes. The GitHub repo must already exist — `codectx publish` handles tagging and validation, not repo creation.

### Published Package codectx.yml

The published `codectx.yml` contains only identity and dependency declarations:

```yaml
name: "react-patterns"
org: "community"
version: "2.3.1"
description: "React component and hook patterns for AI-driven development"

dependencies:
  javascript-fundamentals@community: ">=1.0.0"
```

Published packages do not include `session`, `active` flags, or `registry` configuration. The consumer controls all session context and active/inactive decisions. Published dependencies use semver ranges; the consumer's `codectx.lock` pins exact versions.

### Creating a New Package

Scaffold a new package project:

```bash
codectx new package
```

This creates the standard directory structure for a publishable package, including a CI workflow for releases.

---

## Quality Signal

The compiler itself is the quality contract. A package that compiles cleanly with comprehensive taxonomy coverage, well-structured chunks, and complete manifests is a good package. Compilation reports surface sparse taxonomy, oversized chunks, and validation warnings — quality issues are visible, not hidden.

Each package's content feeds into the project-level taxonomy. Your `system/topics/taxonomy-generation/` instructions govern how aliases are generated for all content uniformly — local and from packages.

---

[Back to overview](README.md)
