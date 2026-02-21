# ElysiaJS Specification

Spec for the ElysiaJS data-layer API conventions. For the conventions themselves, see [README.md](../README.md).

## Purpose

The application needs a data-layer API that handles business logic, data operations, and API-level auth independently of the frontend framework. Without explicit conventions, the API boundary blurs with Next.js server actions, route handlers proliferate without structure, and the API becomes coupled to the frontend's deployment and lifecycle.

## Decisions

- **Embedded via Next.js route handler over standalone server.** A standalone Elysia server (separate Bun process, separate port) adds deployment complexity without current justification. The application does not need independent API scaling. Embedding via a catch-all route handler (`src/app/api/[[...slugs]]/route.ts`) keeps a single process, single port, single deployment. Elysia's WinterCG compliance makes this integration seamless: `app.handle` is a standard `Request -> Response` function that Next.js route handlers accept directly. If the API later needs independent scaling, extracting `src/server/` into a standalone Elysia server is a mechanical refactor because all API logic is already isolated from Next.js. Alternative considered: standalone Elysia server on a separate port (rejected for now; adds infrastructure complexity without a current scaling requirement).

- **Optional catch-all `[[...slugs]]` route.** The double-bracket optional catch-all matches both `/api` (base path) and `/api/anything/else` (nested paths). A single-bracket catch-all (`[...slug]`) would not match the base `/api` path, requiring a separate `route.ts` for the root. The optional variant covers both cases in one file. This is the integration pattern from Elysia's official Next.js documentation.

- **`src/server/` as the API root.** API logic lives outside the Next.js `src/app/` tree to maintain clear separation. `src/api/` was considered but rejected because `src/app/api/` already exists for the Next.js route handler; the naming overlap would cause confusion. `src/server/` is unambiguous: it is where server-side API logic lives.

- **`app.ts` as a direct file, not directory-per-module.** The root Elysia instance is the API entry point, analogous to `src/app/layout.tsx` in Next.js. Entry points do not require the directory-per-module treatment unless the testing strategy justifies colocating `app.test.ts`. Without that justification, a direct file is simpler. Promotable to a directory if testing needs change.

- **Directory-per-module for routes.** Each route is a directory containing its implementation (`[route].ts`), barrel (`index.ts`), and colocated test (`[route].test.ts`). This is the same pattern used for React components and server actions. Applying it to API routes means no new organizational convention to learn. The alternative (route files without directories) would break the colocated test pattern and create an inconsistency with the rest of the codebase.

- **Domain directories with composition files.** Routes are grouped by domain (e.g., `users/`, `health/`). Each domain has a composition file that creates an Elysia instance and composes its routes via `.use()`, plus a barrel that re-exports it. This keeps `app.ts` clean (it only composes domains, not individual routes) and mirrors the recursive composition pattern from the React conventions. Alternative considered: flat route files without domain grouping (rejected; loses cohesion as the API grows and makes it harder to find related routes).

- **Barrel re-export convention for domain composition.** The domain `index.ts` is a barrel (re-export), not a composition file. The composition lives in a named file (`[domain].ts`) that matches its export. This maintains consistency with the rest of the codebase where `index.ts` always means "barrel that re-exports." Alternative considered: using `index.ts` as the composition file (rejected; it would make `index.ts` mean "barrel" everywhere except in Elysia domains, creating a silent inconsistency).

- **Eden Treaty for type-safe API calls.** Eden Treaty eliminates manual `fetch` wrappers and type assertions by inferring all request/response types from the Elysia app's type signature. A single `type App = typeof app` export is the type contract. This aligns with the project's TypeScript conventions (type safety at boundaries) and eliminates an entire category of runtime type errors. Alternative considered: manual fetch with type assertions (rejected; type assertions are not type safety, and maintaining parallel type definitions for every endpoint is error-prone and labor-intensive).

- **Derived type export exception.** `type App = typeof app` is a derived type (it is `typeof` the primary export) that exists solely for Eden Treaty. It is an acceptable exception to one-export-per-module because it carries no independent semantic meaning; it is a mechanical type alias of the value export. Separating it into its own file would add a file with one line of derived type for no practical benefit.

- **Plugin evaluation per Leverage Before Building.** Elysia's plugin system is first-class (`@elysiajs/*` ecosystem). Custom cross-cutting concerns (auth, logging, rate limiting) are common plugin candidates. Applying the [Leverage Before Building](../../../foundation/philosophy.md) principle here means searching for existing plugins before writing custom ones, with the same critical evaluation criteria: maintenance activity, community adoption, bundle size, API quality, security posture.

## Dependencies

- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): Leverage Before Building, Abstractions Must Earn Their Place
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows
- [docs/topics/typescript/README.md](../../typescript/README.md): one-export-per-module, named exports, file naming conventions
- [docs/topics/nextjs/README.md](../../nextjs/README.md): App Router route handlers, server actions boundary
- [docs/topics/nextjs/server-actions.md](../../nextjs/server-actions.md): server action role relative to ElysiaJS
- [docs/topics/nextjs/middleware.md](../../nextjs/middleware.md): page-level vs API-level auth boundary
- `src/app/api/[[...slugs]]/route.ts`: Next.js bridge file (future)
- `src/server/app.ts`: root Elysia instance (future)

## Structure

- `README.md`: ElysiaJS conventions (integration, file organization, Eden Treaty, plugins, separation of concerns)
- `spec/README.md`: this file; reasoning behind the conventions
