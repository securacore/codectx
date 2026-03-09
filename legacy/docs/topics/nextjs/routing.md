# Routing

Routing file conventions, route groups, and dynamic routes in the Next.js App Router. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Routing File Toolkit

The App Router recognizes the following special files. Each serves a distinct purpose. Only create a routing file when the route requires it. Do not create placeholder files speculatively.

| File               | Purpose                                              |
| ------------------ | ---------------------------------------------------- |
| `page.tsx`         | Route UI. Makes a route segment publicly accessible. |
| `layout.tsx`       | Shared UI that persists across child navigations.    |
| `loading.tsx`      | Route-level Suspense fallback (loading state).       |
| `error.tsx`        | Route-level error boundary.                          |
| `not-found.tsx`    | 404 UI for the route segment.                        |
| `global-error.tsx` | Root layout error boundary (one at the app root).    |
| `template.tsx`     | Like layout, but re-mounts on every navigation.      |
| `route.ts`         | API route handler (cannot coexist with `page.tsx`).  |
| `default.tsx`      | Fallback UI for parallel routes.                     |

For the relationship between `error.tsx`/`loading.tsx` and component-level `ErrorBoundary`/`Suspense`, see [docs/topics/react/error-handling.md](../react/error-handling.md).

## Routing File Conventions

<rules>

- Routing files (`page.tsx`, `layout.tsx`, `error.tsx`, `loading.tsx`, `not-found.tsx`, `global-error.tsx`, `template.tsx`, `default.tsx`) use the `function` keyword with default exports. This is a framework requirement, not a style choice. It is the only exception to the arrow function and named export conventions in [docs/topics/react/components.md](../react/components.md).
- Only create routing files when the route needs them. A route does not require `loading.tsx` unless it has async content that benefits from a loading state. A route does not require `error.tsx` unless it has failure modes worth catching at the route level.
- No speculative files. No placeholders. Only what is needed, when it is needed.

</rules>

## Route Organization

Routes live in `src/app/`. Route groups (parenthesized directories) organize routes without affecting the URL structure.

<rules>

- Use route groups to separate authenticated and unauthenticated route sections. Example group names: `(dashboard)`, `(auth)`, `(marketing)`. Specific group names are decided as the application is built.
- Route groups are organizational, not prescriptive. Create them when the application structure calls for them.
- Colocation of non-routing files (components, hooks, features) within route directories is safe. The App Router only recognizes specific filenames (`page.tsx`, `route.ts`, and the special files listed in the toolkit table). All other files and directories are ignored by the router. See [colocation.md](colocation.md) for the full colocation model.

</rules>

## Dynamic Routes

<rules>

- Use `[param]` for dynamic route segments. The parameter name is descriptive: `[projectId]`, `[slug]`, `[invoiceId]`.
- Catch-all routes (`[...param]`) and optional catch-all routes (`[[...param]]`) are context-dependent. Use them when a route genuinely needs to match multiple path segments. There is no hard rule for when to use catch-all vs nested dynamic segments; evaluate per-feature.

</rules>

## API Strategy

The data-layer API is handled by ElysiaJS (see [docs/topics/elysiajs/](../elysiajs/README.md)). Next.js route handlers (`route.ts`) and server actions are both available as tools within the framework.

<rules>

- Do not assume route handlers or server actions as the default API approach. Evaluate per-feature based on two criteria: what the tech stack provides (Next.js, React, ElysiaJS) and what approach optimally achieves the desired goal.
- Route handlers may serve limited or proxy roles given that ElysiaJS handles the data layer. The specific boundary is decided per-feature as the application is built.
- Server action conventions are in [server-actions.md](server-actions.md).

</rules>
