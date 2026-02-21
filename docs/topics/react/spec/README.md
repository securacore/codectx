# React Specification

Spec for the React component conventions. For the conventions themselves, see the [README.md](../README.md) concordance and its linked documents.

## Purpose

React has a vast surface area of patterns, and AI agents will default to whatever patterns dominate their training data. Without explicit conventions, every session produces different component structures, naming patterns, and file organizations. This document captures the reasoning behind the choices so the conventions can be understood, recreated, or revised.

## Decisions

- **Arrow functions only.** Arrow functions work cleanly with the `FC<Props>` type annotation and eliminate stylistic variation. The `function` keyword introduces hoisting behavior and an alternative syntax that adds no value for components. One way to define components means zero decision points. Alternative considered: allowing both styles (rejected; creates inconsistency across files and sessions).

- **`FC<Props>` type annotation.** Provides a consistent component signature, constrains the return type, and works with the global `FC` type from `types/FC.d.ts`. Every component reads the same way. Alternative considered: explicit return type without FC (rejected; more verbose, less consistent, requires importing `ReactNode` for return type). Alternative considered: `React.FC` (rejected; the `React.*` namespace prefix is prohibited per TypeScript conventions).

- **`Props` not `ButtonProps`.** The type is module-scoped and never exported. There is no collision risk. Naming it `Props` eliminates naming ceremony and makes every component's input contract immediately recognizable without reading the type name. The convention is about recognition, not structure. `Props` can be any valid TypeScript type. Alternative considered: component-prefixed names like `ButtonProps` (rejected; redundant when the file is `Button.tsx` and the type is never exported).

- **`Return` for hook output contracts.** Same reasoning as `Props`. The name `Return` is a standardized identifier for the output contract. It is module-scoped, never exported, and immediately recognizable. `Return` is not a reserved word in TypeScript. The type can be any shape (object, tuple, union, primitive). Omit `type Return` when inference is sufficient and the return type is obvious. Alternative considered: `TReturn` (rejected; the `T` prefix is reserved for generic type parameters per TypeScript naming conventions).

- **Recursive component structure.** Components are microcosms of the application. When a component scales, its internal directory mirrors `src/` (components/, hooks/, lib/, providers/). The same organizational principles (one unit per directory, barrels, named exports) apply at every level. This means a developer or AI never learns a different organizational pattern for different scopes. The structure is fractal. Alternative considered: flat internal files (rejected; breaks the one-unit-per-directory convention and creates inconsistency between `src/` level and component level).

- **Internal barrels re-export everything.** Organizational barrels inside a component (`hooks/index.ts`, `components/index.ts`) re-export all their contents as named exports. Everything they expose is consumed by the parent component. Tree-shaking yields the same result whether imports come from the barrel or from individual files. The barrel provides import convenience at zero runtime cost. This is distinct from `src/`-level barrels, which are prohibited because they aggregate unrelated modules where tree-shaking matters.

- **Dot notation namespace for composable components.** Sub-components attached as properties on the parent component provide a single import, clear hierarchy, and self-documenting usage. The `type Component = FC<Props> & { ... }` intersection gives full TypeScript support. Alternative considered: separate named exports for each sub-component (rejected; requires multiple imports, obscures the relationship between components, and violates one-export-per-module). Alternative considered: render props or compound component patterns without namespace (rejected; less discoverable, harder to type).

- **displayName on namespaced components.** React DevTools and error messages use `displayName` to identify components. Without it, namespaced components show as anonymous in the component tree. Setting `displayName` explicitly is required for debuggability.

- **No props drilling.** Passing props through multiple layers where intermediate components only forward them creates tight coupling, defeats render optimization, and makes data flow hard to trace. Intermediate components re-render when props change even if they don't use them. Alternative considered: allowing props drilling with a depth limit (rejected; any limit is arbitrary, and the alternatives are strictly better).

- **State management hierarchy: `useState` first, Jotai second, Context third.** The hierarchy follows a principle of minimal impact. `useState` scopes state to a single component and is the correct default. Jotai provides granular atom-level subscriptions when state needs to exist beyond a single component, ensuring only specific consumers re-render. React Context is for cross-component state where the broad re-render behavior is intentionally correct (the state change fundamentally affects the entire subtree). Each step up the hierarchy widens the re-render blast radius, so escalation requires justification. Alternative considered: Jotai for everything (rejected; `useState` is simpler for local state, and Context's wide-net behavior is the correct tool when an entire subtree should re-render). Alternative considered: Context before Jotai (rejected; Context's lack of granular subscriptions makes it a poor default for shared state, leading to unnecessary re-renders or context-splitting workarounds).

- **Jotai over Zustand.** Jotai's atomic model aligns with the project's bottom-up composition philosophy. Each atom is an independent unit with one reason to exist, matching the one-unit-per-file principle. Atoms compose when needed through derived atoms with explicit, declared dependency graphs. Zustand's flux-based store model structurally tends toward god objects: state lives in a container, and containers grow. The store model requires defining state shape upfront, which conflicts with incremental growth. Relationships between state in a store are implicit (a consequence of co-location), while Jotai atom relationships are explicit and declared. Jotai's atom-level subscriptions are the default behavior; Zustand requires correct selectors to avoid unnecessary re-renders (opt-in granularity vs default granularity). Alternative considered: Zustand (rejected; flux stores structurally tend toward god objects, implicit state relationships, opt-in subscription granularity, and top-down organization misaligns with the project's bottom-up, fractal architecture).

- **React Context for compound component internal state.** Compound components (Disclosure, Tabs, Accordion) need parent-child shared state. Context is the correct mechanism here: the state is private to the component namespace, scoped to a subtree, and the wide-net re-render behavior is intentionally correct because the state change affects all children. Using Jotai atoms for component-internal state would create global state for a local concern.

- **Hook-as-adapter pattern.** Extracting integrations into hooks decouples concerns that change for different reasons. The component changes when the UI changes. The hook changes when the integration changes. The hook's return type (via `type Return`) is the stable contract between them. If the integration is replaced, only the hook implementation changes. This mirrors the Just abstraction pattern: the consumer interacts with a stable interface, and the implementation can change behind it.

- **Extraction triggers are concern-based.** Extract when concerns have different reasons to change, not when a file exceeds some arbitrary size. External integrations, complex state machines, and dense effect groups are integration concerns, not rendering concerns. Isolating them in hooks follows SOLID's single responsibility principle applied at the component level. Alternative considered: size-based thresholds like "extract when a file exceeds 200 lines" (rejected; arbitrary, creates perverse incentives to avoid extraction or to extract prematurely).

- **Single-concern top-level directories.** Each `src/` top-level directory contains only its own kind of unit: `src/components/` for components, `src/hooks/` for hooks, `src/lib/` for utilities, `src/types/` for domain types, `src/providers/` for providers, `src/features/` for global features. No cross-pollination (e.g., `src/components/hooks/` does not exist). This eliminates ambiguity about where a unit lives and enforces the one-concern-per-directory principle at the application level. Alternative considered: grouping by domain at the top level (rejected; conflates concerns and creates directories that mix components, hooks, and utilities).

- **`src/features/` for global domain features.** Global features are domain modules not tied to any specific route: notifications, real-time presence, global search. Each feature has its own recursive internal structure (`components/`, `hooks/`, `lib/`, `providers/`, `actions/`). This provides a clear organizational home for cross-cutting domain concerns without polluting `src/components/` with non-generic code. Alternative considered: placing domain components directly in `src/components/` (rejected; mixes generic UI primitives like Button with domain-specific modules like notifications).

- **Scope through directory location.** Directory location communicates who can consume the code. Feature internals are exclusive to the feature. Route-level directories are shared within the route. `src/`-level directories are application-wide. Dependency flows downward: inner scopes consume from outer scopes, not the reverse. This makes the codebase self-documenting: the file path tells you the scope without reading any configuration or comments.

- **Default values in destructuring.** Providing defaults in the destructuring pattern keeps them colocated with the component signature. The component's API contract (what props exist and what their defaults are) is visible in one place. Alternative considered: `defaultProps` (rejected; deprecated in React 18+). Alternative considered: defaults inside the component body (rejected; separates defaults from the signature, making the contract harder to read at a glance).

- **Server-first component strategy.** Default to Server Components. Add `"use client"` only when interactivity or browser APIs are needed. Full treatment of RSC boundaries and patterns is in [docs/topics/nextjs/server-client.md](../../nextjs/server-client.md). This decision is noted here because it influences the state management hierarchy (Server Components can fetch data where it is needed) and the error handling model (`error.tsx` handles server-side errors at the route level).

- **`react-error-boundary` over custom class component.** React 19 still requires class components for Error Boundaries. Writing a custom class component would be the only exception to the arrow-function-only rule. `react-error-boundary` provides a functional API (`ErrorBoundary` component, `fallbackRender` prop, `useErrorBoundary` hook) that maintains consistency with the rest of the component architecture. The library is small, focused, and well-maintained. Alternative considered: custom class component as a sanctioned exception (rejected; consistency with the functional approach outweighs avoiding a small, stable dependency).

- **Two-tier error and loading model.** Error handling and loading states mirror each other: route-level (`error.tsx`/`loading.tsx`, Next.js managed) and component-level (`ErrorBoundary`/`Suspense`, React managed). Both tiers share the same placement heuristic: isolation boundaries. This symmetry gives a single mental model for both concerns. Alternative considered: separate placement heuristics for errors and loading (rejected; the concerns overlap at the same architectural boundaries, so diverging heuristics adds complexity for no benefit).

- **Isolation boundary placement heuristic.** Error Boundaries and Suspense boundaries go where an independent feature's failure or loading should be contained rather than propagating. This is a concern-based heuristic, not a structural one (not "every component" or "every page section"). The boundary goes where the blast radius of a failure or loading state should stop. Alternative considered: wrapping every async component (rejected; too granular, adds noise). Alternative considered: only route-level boundaries (rejected; too coarse, one widget failure takes down the whole route).

- **Trust the React Compiler for memoization.** The React Compiler automatically inserts the equivalent of `React.memo`, `useMemo`, and `useCallback` based on dependency analysis. Combined with the state management hierarchy (`useState` for local scope, Jotai for granular shared subscriptions, Context for intentional broad re-renders), the optimization surface is almost entirely covered. Manual memoization is acceptable only when profiling identifies a specific bottleneck. Speculative memoization is prohibited because it adds noise, obscures intent, and duplicates work the Compiler already does. Alternative considered: allowing preventive memoization for "obviously expensive" operations (rejected; "obviously expensive" is a judgment call that leads to inconsistent application, and the Compiler handles these cases).

## Dependencies

- `tsconfig.json`: path aliases and strict mode
- `react-error-boundary`: functional Error Boundary component and hooks
- [docs/topics/typescript/README.md](../../typescript/README.md): type system conventions, global types, module design
- [docs/topics/just/README.md](../../just/README.md): operational directive and abstraction pattern
- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): guiding principles (SOLID, one unit per file, abstractions must earn their place)
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows

## Structure

- `README.md`: concordance linking to all React convention documents
- `components.md`: component definition, props, and global types
- `composable-components.md`: dot notation namespace pattern and sub-components
- `file-organization.md`: directory structure, barrels, and recursive pattern
- `hooks.md`: hook conventions and extraction triggers
- `state.md`: state management hierarchy and props drilling
- `error-handling.md`: Error Boundaries, Suspense, and isolation boundaries
- `memoization.md`: React Compiler and manual memoization policy
- `spec/README.md`: this file; reasoning and decisions behind the conventions
