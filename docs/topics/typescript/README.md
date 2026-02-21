# TypeScript

TypeScript conventions and standards for this repository. This document covers the language itself: type design, naming, module structure, and safety patterns. Framework-specific typing (React, Next.js) belongs in the respective framework topic directory.

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Type System

<rules>

- Use `type` for all type definitions. Never use `interface`.
- Never use `enum`. Use string literal unions for type-level sets. Use `as const` objects when runtime values are needed.
- Never use `any`. Use `unknown` for truly unknown values and narrow with type guards.
- Use `readonly` selectively. Apply it only when mutation would be a bug, not as a blanket default.

</rules>

## Type Organization

Types are organized into four tiers by scope. Each tier has distinct rules for location, format, and exports. For the rationale behind this model, see the Four-tier type organization decision in [spec/README.md](spec/README.md).

### Global Types

<rules>

- Location: `types/` at the project root.
- Format: `.d.ts` files. One type per file. PascalCase filename matching the type name.
- Included in `tsconfig.json`. Available globally without imports.
- Purpose: eliminate verbose namespace prefixes.
- If a `React.*` type is used anywhere, create a global type utility instead (e.g., `types/FC.d.ts` for `type FC<P = {}> = React.FC<P>`).
- The `React.*` pattern is prohibited. It is redundant duplication that serves no purpose for the compiler, the IDE, or the developer.

</rules>

### Domain Types

<rules>

- Location: `src/types/`.
- Format: one type per file. PascalCase filename matching the type name.
- A `src/types/index.ts` barrel re-exports all domain types as named exports.
- Domain types represent the application's domain model. They are independent of any feature, component, utility, or specific functionality.

</rules>

### Module Types

<rules>

- Location: within the module file they serve.
- A hook's argument type, a utility's return type. These support the module's implementation.
- Export only if part of the module's public API.

</rules>

### Single-Use Types

<rules>

- Location: within the module file.
- Never exported. They exist as composable building blocks that break complex types into readable pieces.

</rules>

## Module Design

<rules>

- One export per module. Each file exports exactly one thing.
- All exports are named exports.
- Default exports are prohibited. The only exception is when a framework or library requires a default export (e.g., Next.js page/layout components).
- File naming matches the export name. A file exporting `type UserId` is named `UserId.ts`. A file exporting `const apiClient` is named `apiClient.ts`.
- Barrel files (`index.ts`) are permitted at leaf directory boundaries (e.g., `src/lib/foo/index.ts`). Barrels at broad topical directories (e.g., `src/lib/index.ts`) are prohibited.
- Barrel files export only named exports.

</rules>

## Naming Conventions

<rules>

- **Types:** PascalCase. `UserProfile`, `ApiResponse`, `SessionToken`.
- **Variables and functions:** camelCase. `getUserById`, `isActive`, `formatDate`.
- **True constants:** SCREAMING_SNAKE_CASE. Module-level immutable values only. `MAX_RETRIES`, `DEFAULT_TIMEOUT`.
- **Files:** match the export name. PascalCase for type files, camelCase for function/variable files.
- **Generics:** context-dependent. Single letters (`T`, `K`, `V`) for simple utility types. Descriptive names (`TItem`, `TResponse`) for domain-specific generics.

</rules>

## Null and Undefined

<rules>

- `undefined` means uninitialized. The value has not been assigned.
- `null` means initialized but intentionally absent. The value has been explicitly set to "no value."
- This distinction aligns with ECMA semantics and is intentional, not arbitrary.

</rules>

## Type Safety

<rules>

- Exported functions must have explicit return types. They are API contracts.
- Internal functions may rely on type inference.
- Minimize type assertions (`as Type`). `as` is a last resort.
- Prefer type guards, narrowing, and discriminated unions for type safety.
- Use runtime validation (Zod or similar) at external boundaries (API responses, form data, environment variables) where appropriate.
- Strict mode (`strict: true` in `tsconfig.json`) is the baseline. All conventions assume strict mode is active.

</rules>

## Tooling

Formatting and linting are handled by Biome (see `biome.json`). Import organization is handled by Biome's `organizeImports` assist action. The conventions in this document cover semantic patterns that tooling does not enforce.

Compiler configuration is defined in `tsconfig.json`. Per [philosophy.md](../../foundation/philosophy.md), when this document and configuration conflict, configuration wins.
