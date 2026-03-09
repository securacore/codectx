# Middleware

Next.js middleware conventions for request-level concerns. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

<rules>

- Middleware lives in a single `middleware.ts` file at the project root (`src/middleware.ts` when using the `src/` directory). This is a Next.js constraint: only one middleware file is supported.
- Use middleware for request-level concerns: auth gating (redirecting unauthenticated users), redirects, URL rewrites, and response header manipulation.
- Middleware runs before the page renders. It answers "should this page be shown to this user?" not "what data does this page need?"
- Middleware protects page routes. It does not replace API-level authentication or authorization, which is handled by ElysiaJS (see [docs/topics/elysiajs/](../elysiajs/README.md)).
- Specific middleware implementation is decided as the application is built. Do not commit to middleware patterns before the routing structure and auth requirements are established.

</rules>

## Relationship to ElysiaJS

Middleware and ElysiaJS operate at different levels:

- **Next.js middleware:** protects page routes, handles redirects and rewrites for the frontend.
- **ElysiaJS:** handles API-level auth, authorization, data operations, and business rules.

The two complement each other. Middleware gates access to pages; ElysiaJS gates access to data. Neither replaces the other.
