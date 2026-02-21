# Next.js Specification

Spec for the Next.js App Router conventions. For the conventions themselves, see the [README.md](../README.md) concordance and its linked documents.

## Purpose

Next.js provides a large surface area of features (App Router, Server Components, server actions, middleware, route handlers, metadata, caching). Without explicit conventions, AI agents will scaffold routes with speculative files, default to familiar patterns from the Pages Router era, or make premature architectural commitments. This document captures the reasoning behind the choices so the conventions can be understood, recreated, or revised.

## Decisions

- **Self-hosted standalone over Vercel deployment.** The application is deployed independently, not on Vercel. `output: "standalone"` produces a self-contained build that does not depend on `node_modules` at runtime. This keeps the deployment modular and platform-independent. Alternative considered: Vercel deployment (rejected; ties the deployment model to a specific platform, reducing portability).

- **No speculative routing files.** Routing files (`loading.tsx`, `error.tsx`, `not-found.tsx`, etc.) are created only when the route requires them. Speculative files add code with no immediate purpose, create maintenance surface area, and signal intent that may not materialize. Alternative considered: scaffolding a standard set of files for every route (rejected; contradicts the "only what is needed, when it is needed" principle).

- **Merit-based API strategy.** The choice between route handlers (`route.ts`), server actions, and ElysiaJS API calls is made per-feature based on what optimally achieves the desired goal. No single approach is the default. This avoids premature commitment to patterns that may not fit every use case. The decision process is two-fold: first, what the tech stack provides within the framework; second, what approach optimally achieves the goal.

- **Colocation over separation.** Route-specific code lives with the route in `src/app/`, not in separate top-level directories. The App Router's whitelist-based file recognition (only `page.tsx`, `route.ts`, and special convention files become routes) makes this safe. Colocation reduces navigation distance, makes route dependencies visible, and follows the same principle as colocated tests and component internals. Alternative considered: all components in `src/components/` regardless of route affinity (rejected; creates artificial separation, increases navigation distance, and obscures which code belongs to which route).

- **Feature escalation within routes.** Features follow a promotion pattern: feature-exclusive to route-shared to subtree-shared to global. Directory location communicates scope. The promotion trigger is the same at every level: "something outside my scope needs this." This is the same principle as the recursive component structure applied at the route level. Alternative considered: a flat `src/features/` for all features (rejected; loses the scoping information that directory location provides and creates a single directory that aggregates unrelated domain concerns).

- **Route organizational directories mirror src/ structure.** Route directories can contain `components/`, `hooks/`, `lib/`, `providers/`, `actions/`, `features/`, following the same one-concern-per-directory principle as `src/`. This consistency means no new organizational patterns to learn at the route level. Alternative considered: a single `_components` directory per route (rejected; conflates all concerns into one directory, breaks the one-concern-per-directory principle).

- **Server-first with extracted client islands.** Every component is a Server Component by default. Interactive pieces are extracted into the smallest possible `"use client"` component. The server component composes the client island. Starting at the granular level minimizes the client bundle and avoids incurring performance loss from unnecessarily broad client boundaries. Alternative considered: marking pages as `"use client"` when any interactivity is needed (rejected; drags the entire import tree into the client bundle, defeating server-side rendering benefits).

- **Extracted server actions over inline `"use server"`.** Server actions are extracted to their own files in an `actions/` directory, following the one-export-per-module convention. Each action is a single function with `type Props` and `type Return`, consistent with hooks. This makes actions independently testable, discoverable, and refactorable. Alternative considered: inline `"use server"` within Server Components (rejected; mixes concerns, makes actions harder to find and test independently).

- **`actions/` as organizational directory.** Server actions get their own `actions/` directory alongside `components/`, `hooks/`, etc. Actions follow the same colocation and escalation rules as other organizational directories. Alternative considered: placing actions directly in `lib/` (rejected; actions have distinct concerns from utilities, and mixing them obscures what is a server action vs a pure function).

- **ElysiaJS for data-layer API.** The data-layer API is a modular piece not tied to the framework. ElysiaJS handles data operations, business logic, and API-level auth. Next.js handles frontend routing, SSR, and page protection. This separation means the API can evolve independently of the frontend framework. Server actions primarily handle form submissions, cache revalidation, and thin orchestration between the frontend and the API.

- **Middleware for page-level protection.** Next.js middleware handles request-level concerns (auth gating, redirects, headers) at the page level. It does not replace API-level security in ElysiaJS. The two operate at different layers and complement each other. Specific middleware implementation is decided as auth requirements are established.

- **Static metadata as default.** Static `metadata` exports are the default. `generateMetadata` is used only when metadata depends on dynamic data. Every distinct view exports metadata for SEO. Metadata inheritance from parent layouts avoids redeclaring unchanged values. Alternative considered: `generateMetadata` everywhere for consistency (rejected; adds unnecessary async overhead for static content).

- **Turbopack for development.** Turbopack is the development bundler, configured via `next dev --turbopack`. It provides faster development builds than webpack. This is a configuration decision, not a convention that affects application code.

- **React Compiler via config.** The React Compiler is enabled through `reactCompiler: true` in `next.config.ts` and the `babel-plugin-react-compiler` dev dependency. This is the mechanism that enables the trust-the-compiler memoization convention in [docs/topics/react/memoization.md](../../react/memoization.md).

## Dependencies

- `next.config.ts`: standalone output, React Compiler, Turbopack
- `components.json`: shadcn/ui RSC configuration
- [docs/topics/react/file-organization.md](../../react/file-organization.md): recursive pattern, scope/ownership principles, application-level organization
- [docs/topics/react/error-handling.md](../../react/error-handling.md): two-tier error/loading model
- [docs/topics/react/components.md](../../react/components.md): routing file exception (function + default export)
- [docs/topics/react/hooks.md](../../react/hooks.md): Props/Return naming convention (applied to server actions)
- [docs/topics/typescript/README.md](../../typescript/README.md): one-export-per-module, named exports, module design
- [docs/topics/just/README.md](../../just/README.md): operational directive (all commands through Just)
- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): guiding principles
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows

## Structure

- `README.md`: concordance linking to all Next.js convention documents
- `routing.md`: routing files, route groups, dynamic routes, API strategy
- `colocation.md`: route colocation, feature escalation, scope model
- `server-client.md`: Server Components, Client Components, use client boundary
- `server-actions.md`: server action conventions, actions/ directory, examples
- `metadata.md`: static and dynamic metadata for SEO
- `middleware.md`: request-level auth gating, redirects, ElysiaJS relationship
- `configuration.md`: next.config.ts, standalone mode, development tooling
- `spec/README.md`: this file; reasoning and decisions behind the conventions
