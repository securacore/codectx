# Package Format

Every documentation package follows the same structure, whether it is the project's local documentation or an installed dependency. A package is a directory containing a `package.yml` manifest and optional documentation directories.

## Structure

```text
[package]/
  package.yml
  foundation/
  topics/
  prompts/
  plans/
    [plan-name]/
      README.md
      state.yml
```

Not every package contains all directories. A package may provide only topics, or only prompts, or any combination.

## The Data Map (`package.yml`)

Every package has a `package.yml` at its root. This file is the data map: a navigation index that tells AI what documentation exists and how to load it. It is validated by [package.schema.json](../schemas/package.schema.json).

`package.yml` contains:

- **Package metadata**: name, author, version, description
- **Documentation entries** organized into four sections: foundation, topics, prompts, plans

Each entry in the data map has:

- A unique ID
- A file path relative to the package root
- A description
- Dependency relationships (`depends_on` / `required_by`)

### Foundation Entries

Foundation entries have a `load` field that controls when they are loaded into AI context:

- `always` -- loaded at the start of every session
- `documentation` -- loaded only when the task involves documentation work

### Plan Entries

Plan entries have a `state` field pointing to a `state.yml` file for lightweight status tracking. AI reads `state.yml` first to assess plan status without loading the full plan document.

## Naming and Resolution

Packages are identified by name and author. The naming convention is `name@author`.

| Format | Example | Meaning |
|---|---|---|
| `name` | `react` | Latest version, unscoped |
| `name:version` | `react:^1.0.0` | Versioned, unscoped |
| `name@author` | `react@org` | Latest version, scoped |
| `name@author:version` | `react@org:^1.0.0` | Fully qualified |

The `codectx-` prefix is always implied. `react@org` resolves to the `codectx-react` repository owned by `org` on GitHub.

Versions follow semver. Range syntax is supported: `^1.0.0` (compatible), `~1.0.0` (patch-level), `1.0.0` (exact). Versions are resolved from Git tags in semver format (e.g., `v1.0.0`).

Package resolution is Git-first. Packages are fetched from Git repositories. The source URL is either specified explicitly in `codectx.yml` or inferred from the name and author. The naming convention is designed to support a future package registry without breaking changes.

## Plans and State Tracking

Plans are implementation documents that describe what to build and how. Each plan is a directory containing `README.md` (the plan content) and `state.yml` (the state tracker).

`state.yml` is validated by [state.schema.json](../schemas/state.schema.json) and contains:

- Plan ID
- Status: `not_started`, `in_progress`, `completed`, `blocked`
- Optional timestamps: `started_at`, `updated_at`
- Summary: one to three sentences describing current state

AI reads `state.yml` first to triage whether to load the full plan. This avoids loading large plan documents unnecessarily.

## Related

- [Configuration](configuration.md) -- how packages are declared and activated in `codectx.yml`
- [Compilation](compilation.md) -- how packages are compiled into the output format
- [Design Decisions](spec/README.md) -- reasoning behind package format choices
