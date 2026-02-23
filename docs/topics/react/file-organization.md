# File Organization

Directory structure, barrel conventions, and the recursive component pattern. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

<rules>

- Every unit (component, hook, utility, provider) lives in its own named directory with its own `index.ts` barrel.
- The filename matches the unit name. `Button/Button.tsx`, `useCardApi/useCardApi.ts`, `formatCardData/formatCardData.ts`.
- No flat files in organizational directories. A hook is never a bare file inside `hooks/`. It is always `hooks/useHookName/useHookName.ts`.
- Start minimal. A simple component is just the component file and a barrel. Add internal structure only when needed.
- Test files are colocated with their unit (`Button.test.tsx`). Testing conventions are in `docs/topics/testing/` (future).
- Story files are colocated with their unit (`Button.stories.tsx`). Storybook is planned but not yet configured.

</rules>

## Minimal Component

```text
Button/
  Button.tsx
  Button.test.tsx
  Button.stories.tsx
  index.ts
```

### Incorrect: Flat Files

```text
# WRONG: hooks as bare files in hooks/
hooks/
  useCardApi.ts           # must be hooks/useCardApi/useCardApi.ts
  useCardState.ts         # must be hooks/useCardState/useCardState.ts

# WRONG: component without its own directory
components/
  Header.tsx              # must be components/Header/Header.tsx
```

## Scaled Component

When a component needs sub-components, hooks, utilities, or providers, its internal directory mirrors the `src/` structure. Every internal subdirectory follows the same conventions: one unit per directory, barrels, named exports.

```text
Card/
  Card.tsx
  Card.test.tsx
  Card.stories.tsx
  index.ts
  components/
    Header/
      Header.tsx
      Header.test.tsx
      index.ts
    Body/
      Body.tsx
      Body.test.tsx
      index.ts
    Footer/
      Footer.tsx
      Footer.test.tsx
      index.ts
      components/
        FooterAction/
          FooterAction.tsx
          FooterAction.test.tsx
          index.ts
    index.ts
  hooks/
    useCardApi/
      useCardApi.ts
      useCardApi.test.ts
      index.ts
    useCardState/
      useCardState.ts
      index.ts
    index.ts
  lib/
    formatCardData/
      formatCardData.ts
      index.ts
    index.ts
```

## Barrel Conventions

<rules>

- Leaf barrels re-export the single named export of their unit. `Button/index.ts` exports `{ Button }`.
- Internal organizational barrels (`components/index.ts`, `hooks/index.ts`, `lib/index.ts`) re-export all their contents as named exports. Everything they expose is consumed by the parent component, so tree-shaking yields the same result.
- The component's top-level barrel (`Card/index.ts`) exports only the assembled component (single named export). This is the public API.
- `src/`-level broad barrels remain prohibited per [TypeScript conventions](../typescript/README.md).
- Colocated hooks and utilities are private to the component. They are not re-exported through the component's top-level barrel.

</rules>

```typescript
// Correct: leaf barrel re-exports the single named export
// Button/index.ts
export { Button } from "./Button";

// Correct: top-level barrel exports only the assembled component
// Card/index.ts
export { Card } from "./Card";

// Incorrect: top-level barrel re-exporting internals
// Card/index.ts
export { Card } from "./Card";
export { useCardApi } from "./hooks";    // WRONG: internal hook exposed publicly
export { Header } from "./components";   // WRONG: sub-component exposed directly

// Incorrect: broad barrel at src/ level
// src/components/index.ts
export { Button } from "./Button";       // WRONG: prohibited per TypeScript conventions
export { Card } from "./Card";
```

## Application-Level Organization

The `src/` directory uses the same one-concern-per-directory principle at the top level. Each directory contains only its own kind of unit.

<rules>

- `src/components/` contains shared React components only. No hooks, utilities, or other concerns.
- `src/hooks/` contains shared hooks only. No components or utilities.
- `src/lib/` contains shared utility functions only.
- `src/types/` contains domain types only (see [TypeScript conventions](../typescript/README.md)).
- `src/providers/` contains shared provider components only.
- `src/features/` contains global features: domain modules that are not tied to any specific route. Each feature has its own recursive internal structure (`components/`, `hooks/`, `lib/`, `providers/`, `actions/`).
- No cross-pollination. `src/components/hooks/` does not exist. A hook either lives inside a component's recursive structure (private to that component) or in `src/hooks/` (globally shared).

</rules>

```text
src/
  components/           # shared React components (Button, Card, etc.)
    Button/
      Button.tsx
      index.ts
  hooks/                # shared hooks
    useToggle/
      useToggle.ts
      index.ts
  lib/                  # shared utilities
    formatDate/
      formatDate.ts
      index.ts
  types/                # domain types
    UserId.ts
  providers/            # shared providers
    ThemeProvider/
      ThemeProvider.tsx
      index.ts
  features/             # global features (not route-specific)
    notifications/
      components/
      hooks/
      lib/
      index.ts
```

### Incorrect: Cross-Pollination

```text
# WRONG: hooks inside components directory
src/components/
  hooks/                  # hooks belong in src/hooks/, not inside src/components/

# WRONG: utilities inside hooks directory
src/hooks/
  lib/                    # utilities belong in src/lib/, not inside src/hooks/

# WRONG: domain component in generic components
src/components/
  NotificationPanel/      # domain-specific; belongs in src/features/notifications/
```

### Scope and Ownership

Directory location communicates who can consume the code. Dependency flows downward: inner scopes can consume from outer scopes, but outer scopes never reach into inner scope internals.

<rules>

- A feature's internal directories (`features/[feature]/components/`, `features/[feature]/hooks/`) are exclusive to that feature. They are not consumed outside the feature boundary.
- A feature's barrel (`index.ts`) is its public API. External consumers import from the barrel, not from internals.
- `src/`-level directories are available to the entire application.
- The promotion trigger is the same at every level: "something outside my scope needs this." Move the code to the nearest scope that contains all consumers.

</rules>

```typescript
// Correct: importing from a feature's public API
import { notify } from "@/features/notifications";

// Incorrect: reaching into a feature's internals
import { NotificationBanner } from "@/features/notifications/components/NotificationBanner";
// WRONG: bypasses the feature's barrel, couples to internal structure
```

For how this pattern applies to route-level organization in the Next.js App Router, see [docs/topics/nextjs/colocation.md](../nextjs/colocation.md).
