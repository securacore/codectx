# Configuration

Next.js configuration conventions. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## next.config.ts

<rules>

- `output: "standalone"` is required. The application is self-hosted, not deployed on Vercel. Standalone mode produces a self-contained build that does not depend on `node_modules` at runtime.
- `reactCompiler: true` is required. The React Compiler is enabled for automatic memoization (see [docs/topics/react/memoization.md](../react/memoization.md)).
- Keep `next.config.ts` minimal. Configuration that varies by environment (API URLs, feature flags, secrets) goes in environment variables, not in the config file.
- Use the TypeScript config format (`.ts`). Do not use `.js` or `.mjs`.

</rules>

## Development

<rules>

- Turbopack is the development bundler (`next dev --turbopack`). It is configured in `package.json` scripts.
- All development commands run through Just. Do not invoke `next`, `bun`, or other CLI tools directly. See [docs/topics/just/README.md](../just/README.md).

</rules>

## Related Configuration

- `components.json` (shadcn/ui) is configured with `rsc: true` for React Server Component compatibility. Full shadcn conventions are in `docs/topics/shadcn/` (future).
- `tsconfig.json` includes the Next.js TypeScript plugin, path aliases (`@/*` mapping to `src/*`), and strict mode. See [docs/topics/typescript/README.md](../typescript/README.md).
- `postcss.config.mjs` configures Tailwind CSS 4 via `@tailwindcss/postcss`. Full Tailwind conventions are in [docs/topics/tailwind/README.md](../tailwind/README.md).
