# TypeScript Specification

Spec for the TypeScript conventions documentation. For the conventions themselves, see [README.md](../README.md).

## Purpose

TypeScript has configuration, tooling, and language-level conventions that must be codified so AI agents and engineers produce consistent code. Without explicit standards, every session reinvents type patterns, naming rules, module structure, and null-handling semantics.

## Decisions

- **Scope: language conventions only.** Framework-specific typing (React, Next.js) belongs in respective topic directories. This keeps TypeScript docs framework-agnostic and avoids updating them when a framework convention changes. Alternative considered: including framework typing (rejected; mixes concerns).

- **`type` only, never `interface`.** Types are more versatile (unions, intersections, mapped types, conditional types) and eliminate decision fatigue. Alternative considered: interface for objects, type for the rest (rejected; creates an unnecessary decision point at every type definition).

- **No enums.** Enums generate runtime JavaScript, have quirky behavior (reverse mapping, nominal typing), and add unnecessary overhead. String literal unions and `as const` objects accomplish the same goals with zero runtime cost. Alternative considered: `const enum` (rejected; bundler compatibility issues).

- **Zero `any`.** Absolute rule. `unknown` with type guards replaces every use case where `any` would appear.

- **Explicit return types on exports, inferred on internals.** Exported functions are API contracts where accidental signature changes must be caught. Internal functions benefit from inference to reduce noise.

- **Barrel exports at leaf directories only.** Broad barrels defeat tree-shaking and obscure the dependency graph. Leaf-level barrels provide import convenience without pathological issues. Alternative considered: no barrels at all (rejected; leaf-level barrels are genuinely useful).

- **Named exports exclusively.** Default exports are prohibited unless forced by framework. Named exports enable tree-shaking and make the dependency graph explicit.

- **One export per module.** Unix philosophy: one module does one thing. Maximizes composability, simplifies the dependency graph, makes each file's purpose unambiguous.

- **File naming matches the export.** 1:1 mapping between file names and contents. No conventions to remember beyond "the file is named what it exports."

- **SCREAMING_SNAKE_CASE for true constants.** Distinguishes compile-time immutable values from runtime-computed values. PascalCase is reserved for types, camelCase for everything else.

- **Four-tier type organization.** Types at different scopes have different needs: global utilities need `.d.ts` for zero-import availability, domain types need a barrel for discoverability, module types need colocation for cohesion, single-use types need isolation for readability. Four tiers map to four distinct sets of rules. See the Type Organization section in [README.md](../README.md) for the rules themselves. Alternative considered: single `types/` directory for everything (rejected; mixes scopes). Alternative considered: colocate all types with implementation (rejected; global and domain types are shared and have no single home module).

- **Global types eliminate namespace prefixes.** The `React.*` pattern is redundant duplication. The compiler and IDE do not need it. Global `.d.ts` types remove the prefix at the source.

- **`undefined` for uninitialized, `null` for initialized-but-no-value.** Aligns with ECMA semantics. Both have a place; the distinction is intentional.

- **Minimize assertions, prefer guards.** `as` overrides the compiler's judgment and hides bugs. Type guards, narrowing, and discriminated unions prove type safety rather than asserting it.

- **Selective `readonly`.** Blanket `readonly` adds noise without proportional safety gains. Use it when mutation would be a bug. Alternative considered: liberal readonly by default (rejected; noise without proportional value).

- **Context-dependent generic naming.** Simple utility types don't benefit from verbose names. Domain generics do. Readability at the point of use governs.

- **Configuration as source of truth.** The conventions doc references `tsconfig.json` and `biome.json` rather than duplicating settings. Biome handles what it handles; the docs cover what it doesn't. Follows the Configuration is Truth principle from [philosophy.md](../../../foundation/philosophy.md).

- **Strict mode as baseline.** `tsconfig.json` has `strict: true`. No documentation about enabling it. It is a given.

## Dependencies

- `tsconfig.json`: compiler configuration
- `biome.json`: linting and formatting rules
- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): guiding principles
- [docs/foundation/ai-authoring.md](../../../foundation/ai-authoring.md): cross-model authoring conventions
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows

## Structure

- `README.md`: TypeScript conventions (the actionable rules)
- `spec/README.md`: this file; reasoning and decisions behind the conventions
