# Colocation

Route colocation and the feature escalation path within `src/app/`. For the general principles of file organization, scope, and ownership, see [docs/topics/react/file-organization.md](../react/file-organization.md). For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Colocation Safety

The Next.js App Router only recognizes specific filenames as routes: `page.tsx`, `route.ts`, and the special convention files (`layout.tsx`, `error.tsx`, `loading.tsx`, `not-found.tsx`, `global-error.tsx`, `template.tsx`, `default.tsx`). All other files and directories inside `src/app/` are ignored by the router.

<rules>

- Any directory can be safely created inside a route segment without becoming a route. `components/`, `hooks/`, `lib/`, `providers/`, `actions/`, `features/` are all safe.
- The only way a directory becomes a route segment is by containing a `page.tsx` or `route.ts`. Do not accidentally place these files inside organizational directories.

</rules>

## Organizational Directories in Routes

Route directories follow the same recursive pattern as scaled components. When a route needs its own components, hooks, utilities, providers, or features, it contains organizational directories that mirror the `src/` structure.

```text
src/app/(dashboard)/projects/
  page.tsx
  loading.tsx
  components/
    ProjectTable/
      ProjectTable.tsx
      index.ts
  hooks/
    useProjectFilters/
      useProjectFilters.ts
      index.ts
  providers/
    ProjectFilterProvider/
      ProjectFilterProvider.tsx
      index.ts
  features/
    project-stats/
      components/
      hooks/
      index.ts
  actions/
    createProject/
      createProject.ts
      index.ts
```

## Feature Escalation Within Routes

Features within routes follow the same scope and promotion pattern defined in [docs/topics/react/file-organization.md](../react/file-organization.md). Directory location communicates who can consume the code.

<rules>

- **Feature-exclusive code:** `[route]/features/[feature]/components/` is exclusive to that feature. It is not consumed by the route's `page.tsx`, sibling features, or anything outside the feature.
- **Route-shared code:** `[route]/components/` is available to everything in that route, including nested features. Code lives here when multiple features within the route need it, or when the page itself needs it alongside features.
- **Subtree-shared code:** `[common-ancestor]/components/` is available to all routes within that subtree. Code promotes here when multiple child routes need it.
- **Global features:** `src/features/` is for features that have no route domain dependency. See [docs/topics/react/file-organization.md](../react/file-organization.md).
- **Application-wide shared code:** `src/components/`, `src/hooks/`, `src/lib/` are domain-agnostic. See [docs/topics/react/file-organization.md](../react/file-organization.md).

</rules>

### Escalation Example

A `ProjectCard` component starts inside a feature:

```text
# Step 1: exclusive to the project-stats feature
src/app/(dashboard)/projects/features/project-stats/components/ProjectCard/

# Step 2: a sibling feature also needs it; promote to route level
src/app/(dashboard)/projects/components/ProjectCard/

# Step 3: the invoices route also needs it; promote to common ancestor
src/app/(dashboard)/components/ProjectCard/

# Step 4: unauthenticated marketing pages also need it; promote to src/
src/components/ProjectCard/
```

The promotion trigger is always: "something outside my scope needs this." Move the code to the nearest scope that contains all consumers.

### Incorrect Patterns

```text
# WRONG: feature internals consumed by the route's page.tsx
src/app/(dashboard)/projects/features/project-stats/components/ProjectCard/
  # imported by src/app/(dashboard)/projects/page.tsx -- violates feature encapsulation

# WRONG: cross-pollination at route level
src/app/(dashboard)/projects/components/hooks/
  # hooks directory inside components -- each concern gets its own directory
```

## Scope Summary

| Location | Available to |
| --- | --- |
| `features/[feature]/components/` | Only that feature |
| `[route]/components/` | Everything in that route, including its features |
| `[common-ancestor]/components/` | Everything in that route subtree |
| `src/features/[feature]/` | Internal to that global feature |
| `src/components/` | Entire application |
