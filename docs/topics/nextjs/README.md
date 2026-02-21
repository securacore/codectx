# Next.js

Next.js 16 App Router conventions for this repository. This is a self-hosted application deployed in standalone mode, not on Vercel. Next.js owns frontend routing, server-side rendering, and page protection. The data-layer API is handled by ElysiaJS; conventions for that are in [docs/topics/elysiajs/](../elysiajs/README.md).

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

| Document                                     | Purpose                                                 |
| -------------------------------------------- | ------------------------------------------------------- |
| [routing.md](routing.md)                     | Routing files, route groups, and dynamic routes         |
| [colocation.md](colocation.md)               | Route colocation and feature escalation within src/app/ |
| [server-client.md](server-client.md)         | Server Components, Client Components, use client        |
| [server-actions.md](server-actions.md)       | Server action conventions and file organization         |
| [metadata.md](metadata.md)                   | Static and dynamic metadata for SEO                     |
| [middleware.md](middleware.md)                | Request-level auth gating, redirects, and headers       |
| [configuration.md](configuration.md)         | next.config.ts, standalone mode, and environment        |
