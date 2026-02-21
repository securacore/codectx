# ElysiaJS

ElysiaJS is the data-layer API for this application. It handles data operations, business logic, and API-level authentication. ElysiaJS is embedded within Next.js via a catch-all route handler, running in the same process. Next.js owns frontend routing, server-side rendering, and page protection. ElysiaJS owns the API. The two share a process but not responsibilities.

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Integration

ElysiaJS is mounted inside a Next.js App Router catch-all route handler. The route handler is a thin bridge: it imports the Elysia app instance and exports its `handle` method for each HTTP method. All API logic lives in `src/server/`, not in the route handler.

<rules>

- The bridge file lives at `src/app/api/[[...slugs]]/route.ts`. This is the only file in `src/app/` that touches ElysiaJS. It contains no business logic.
- The Elysia app instance lives in `src/server/app.ts` with `prefix: '/api'` to match the route handler path.
- The bridge file exports the Elysia `handle` method for every HTTP method the API supports (`GET`, `POST`, `PUT`, `PATCH`, `DELETE`). These multiple exports are a Next.js framework requirement for route handlers.
- `src/server/app.ts` exports `type App = typeof app` alongside the app instance. This derived type export is an acceptable exception to the one-export-per-module rule. Eden Treaty requires it for end-to-end type inference.

</rules>

```typescript
// Correct: thin bridge file (src/app/api/[[...slugs]]/route.ts)
import { app } from "@/server/app";

export const GET = app.handle;
export const POST = app.handle;
export const PUT = app.handle;
export const PATCH = app.handle;
export const DELETE = app.handle;
```

```typescript
// Correct: Elysia app instance (src/server/app.ts)
import { Elysia } from "elysia";
import { users } from "./routes/users";
import { health } from "./routes/health";

export const app = new Elysia({ prefix: "/api" })
  .use(users)
  .use(health);

export type App = typeof app;
```

```typescript
// Incorrect: business logic in the bridge file
import { Elysia, t } from "elysia";

const app = new Elysia({ prefix: "/api" })
  .get("/users", async () => {
    // WRONG: API logic belongs in src/server/, not in the route handler
  });

export const GET = app.handle;
```

## File Organization

All ElysiaJS code lives in `src/server/`. This directory is the API boundary, separate from the Next.js `src/app/` tree. The internal structure follows the same directory-per-module convention used by React components and server actions: every module is a directory containing its implementation, barrel export, and colocated test.

<rules>

- `src/server/app.ts` is the root Elysia instance. It composes domain route groups via `.use()`. This is a direct file (entry point), not a directory-per-module. If the testing strategy later requires `app.test.ts`, promote it to a directory.
- Route modules live in `src/server/routes/`, organized by domain (e.g., `users/`, `health/`, `projects/`).
- Each domain directory contains a composition file (`[domain].ts`) that creates an Elysia instance and composes its routes via `.use()`, and a barrel (`index.ts`) that re-exports from the composition file.
- Each individual route is a directory following the directory-per-module convention: `[route]/[route].ts` (implementation), `[route]/index.ts` (barrel), `[route]/[route].test.ts` (colocated test).
- Each route file exports one named Elysia instance with one route definition. File name matches the export name (camelCase), consistent with the [TypeScript naming conventions](../typescript/README.md).
- Shared domain types live in a direct `types.ts` file within the domain directory. Shared validation schemas live in a direct `schemas.ts` file. These are shared resources, not standalone modules.
- `src/server/lib/` contains server-side utilities. Modules follow directory-per-module.
- `src/server/plugins/` contains cross-cutting Elysia plugins (auth, logging, rate limiting). Modules follow directory-per-module.

</rules>

```text
src/server/
  app.ts                                # Root Elysia instance (direct file, entry point)
  routes/
    health/
      index.ts                          # Domain barrel: export { health } from './health'
      health.ts                         # Composition: export const health
      check/
        index.ts                        # Route barrel: export { check } from './check'
        check.ts                        # Route: export const check
        check.test.ts                   # Colocated test
    users/
      index.ts                          # Domain barrel: export { users } from './users'
      users.ts                          # Composition: export const users
      listUsers/
        index.ts                        # Route barrel: export { listUsers } from './listUsers'
        listUsers.ts                    # Route: export const listUsers
        listUsers.test.ts               # Colocated test
      getUserById/
        index.ts                        # Route barrel: export { getUserById } from './getUserById'
        getUserById.ts                  # Route: export const getUserById
        getUserById.test.ts             # Colocated test
      createUser/
        index.ts                        # Route barrel: export { createUser } from './createUser'
        createUser.ts                   # Route: export const createUser
        createUser.test.ts              # Colocated test
      types.ts                          # Shared domain types
      schemas.ts                        # Shared validation schemas
  lib/                                  # Server-side utilities (directory-per-module)
  plugins/                              # Cross-cutting Elysia plugins (directory-per-module)
```

```typescript
// Correct: domain composition file (src/server/routes/users/users.ts)
import { Elysia } from "elysia";
import { listUsers } from "./listUsers";
import { getUserById } from "./getUserById";
import { createUser } from "./createUser";

export const users = new Elysia({ prefix: "/users" })
  .use(listUsers)
  .use(getUserById)
  .use(createUser);
```

```typescript
// Correct: domain barrel (src/server/routes/users/index.ts)
export { users } from "./users";
```

```typescript
// Correct: individual route (src/server/routes/users/listUsers/listUsers.ts)
import { Elysia } from "elysia";

export const listUsers = new Elysia()
  .get("/", () => {
    // data operations
  });
```

```typescript
// Correct: route barrel (src/server/routes/users/listUsers/index.ts)
export { listUsers } from "./listUsers";
```

```typescript
// Incorrect: all routes in one file
import { Elysia } from "elysia";

// WRONG: violates one-export-per-module. Each route gets its own directory.
export const users = new Elysia({ prefix: "/users" })
  .get("/", () => { /* list */ })
  .get("/:id", ({ params }) => { /* get by id */ })
  .post("/", ({ body }) => { /* create */ });
```

```typescript
// Incorrect: route file without directory-per-module
// src/server/routes/users/listUsers.ts  // WRONG: must be listUsers/listUsers.ts
```

## Eden Treaty

Eden Treaty provides end-to-end type-safe API calls from the frontend. The client infers all request/response types from the Elysia app's type signature, eliminating manual type assertions and fetch wrappers.

<rules>

- Use Eden Treaty (`@elysiajs/eden`) for all API calls from the frontend. Do not use manual `fetch` with type assertions.
- The client is created from `type App` exported by `src/server/app.ts`. This is the single source of type truth for the API.
- Eden Treaty calls return `{ data, error, status }`. Handle the `error` case explicitly; do not assume success.

</rules>

```typescript
// Correct: Eden Treaty client setup
import { treaty } from "@elysiajs/eden";
import type { App } from "@/server/app";

const api = treaty<App>("http://localhost:3000");

// Correct: type-safe API call with error handling
const { data, error } = await api.api.users.get();
if (error) {
  // handle error (fully typed by status code)
} else {
  // data is fully typed: Array<{ id: number, name: string, ... }>
}
```

```typescript
// Incorrect: manual fetch with type assertion
const response = await fetch("/api/users");
const data = (await response.json()) as User[];  // WRONG: use Eden Treaty. Type assertion is not type safety.
```

## Plugin Conventions

Elysia's plugin system is first-class. Plugins are Elysia instances composed via `.use()`, the same mechanism used for route composition. Evaluate existing Elysia plugins before writing custom solutions, following the [Leverage Before Building](../../foundation/philosophy.md) principle.

<rules>

- Search for existing Elysia plugins (`@elysiajs/*` and community packages) before writing custom middleware or cross-cutting concerns.
- Evaluate plugin candidates critically: maintenance activity, community adoption, bundle size, API quality, security posture, and alignment with the project's conventions.
- Custom plugins live in `src/server/plugins/`, following directory-per-module. Each plugin exports a named Elysia instance.
- Plugin evaluation follows the same scrutiny as design system exceptions: the AI actively searches for alternatives and pushes back on custom solutions when a well-maintained plugin exists.

</rules>

## Separation of Concerns

ElysiaJS and Next.js operate at different layers. The boundary is clear and intentional.

<rules>

- **ElysiaJS:** data operations, business logic, API-level auth and authorization, validation.
- **Next.js:** frontend routing, server-side rendering, page protection via middleware.
- **Server actions:** form submissions, cache revalidation, thin orchestration between frontend and API. Server actions are not the data access layer.
- The specific boundary between server actions and ElysiaJS API calls is decided per-feature.

</rules>

For the Next.js side of this boundary, see [Next.js server actions](../nextjs/server-actions.md) and [Next.js middleware](../nextjs/middleware.md).

### Future Sections

Auth strategy, database integration, error handling patterns, and API testing conventions are documented as the application is built. These sections depend on implementation decisions that are made per-feature.

## Key Constraints

<rules>

- All API logic in `src/server/`. The Next.js bridge file is a thin pass-through.
- Directory-per-module for every route: `[route]/[route].ts` + `[route]/index.ts` + `[route]/[route].test.ts`.
- One route per file. One named export per route file.
- Eden Treaty for all frontend API calls. No manual fetch with type assertions.
- Evaluate existing plugins before writing custom solutions.

</rules>
