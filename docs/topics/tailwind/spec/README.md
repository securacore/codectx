# Tailwind Specification

Spec for the Tailwind CSS conventions. For the conventions themselves, see [README.md](../README.md).

## Purpose

Tailwind is the styling system for the application. These conventions establish how the design token system works, how utilities are used, and how the design system's integrity is maintained. Without them, styling decisions would be ad hoc, design tokens would proliferate without structure, and the utility-first discipline would erode as the codebase grows.

## Decisions

- **Tailwind 4 CSS-first configuration.** All configuration lives in `src/app/globals.css` via `@theme`, `@custom-variant`, and `@import` directives. There is no JavaScript config file. This aligns with Tailwind 4's design direction, reduces tooling surface, and keeps configuration colocated with the CSS it governs. Alternative considered: maintaining a `tailwind.config.ts` alongside CSS config (rejected; Tailwind 4 made the JS config optional specifically to consolidate configuration into CSS. Maintaining both creates a split source of truth).

- **oklch color space for design tokens.** All color values use `oklch()`. oklch is perceptually uniform, meaning equal numeric changes produce equal perceived changes in lightness, chroma, and hue. This makes the color system more predictable for programmatic manipulation and ensures consistent contrast ratios across the palette. Alternative considered: hex or rgb values (rejected; neither is perceptually uniform, and oklch is the direction the CSS specification and Tailwind ecosystem are moving).

- **CSS custom properties as the theming mechanism.** Design tokens are CSS custom properties (`:root` for light, `.dark` for dark) mapped to Tailwind tokens via `@theme inline`. This creates a single source of truth for design values with automatic dark mode support through property override. Alternative considered: Tailwind 4's `@theme` without CSS custom properties (rejected; CSS custom properties enable runtime theme switching and integration with component libraries).

- **Class-based dark mode.** Runtime theme switching requires programmatic control via JavaScript, independent of the operating system's color scheme preference. Alternative considered: media-query-based dark mode (rejected; the application needs explicit theme control, not just OS preference detection).

- **Utility-first with rigorous exception policy.** Direct application of the [Abstractions Must Earn Their Place](../../../foundation/philosophy.md) principle. The design system's token set and Tailwind's utility library cover virtually all styling needs; the exception bar is intentionally high to prevent erosion over time. Alternative considered: strict zero-exception policy (rejected; acknowledging that edge cases exist while making the bar extremely high is more durable than an absolute ban that eventually gets violated without documentation).

- **`cn()` and `cva()` as the class composition stack.** `cn()` (`clsx` + `tailwind-merge`) handles conditional class merging with intelligent deduplication. `cva()` (`class-variance-authority`) handles component variant APIs with type safety. Both are project standards. Alternative considered: manual template literals and conditional logic (rejected; template literals don't handle class conflicts, and conditional logic becomes unreadable at scale. `cn()` and `cva()` solve distinct problems and compose cleanly together).

- **Mobile-first responsive design.** Build for mobile, layer up for larger viewports using Tailwind's `min-width` breakpoints. This aligns with Tailwind's default breakpoint system and positions the application for future CapacitorJS wrapping as a mobile application. Alternative considered: desktop-first with responsive scaling down (rejected; mobile-first produces simpler base styles and aligns with the progressive enhancement model. Desktop-first requires overriding more styles at smaller breakpoints, which is harder to maintain).

- **shadcn as bootstrap, not architectural commitment.** The initial design token set and component library come from shadcn/ui (new-york style, neutral base, CSS variables mode). shadcn provides a strong starting point with modern industry standards. Over time, components will be graduated into the project's own design system, detaching from the shadcn lifecycle. Once a component is fully adapted to the project's design system and conventions, it becomes an independently maintained component with no shadcn dependency. Alternative considered: building the component library from scratch (rejected for initial velocity; shadcn provides a well-engineered starting point). Alternative considered: permanently adopting shadcn's lifecycle (rejected; shadcn's design decisions will eventually diverge from the project's design system, creating maintenance friction).

- **AI as design system gatekeeper.** Less experienced engineers may not know the design system deeply enough to evaluate whether an exception is truly necessary, while the AI can systematically check every alternative path. This creates a quality gate that scales with team experience without requiring senior review for every styling decision.

## Dependencies

- [docs/foundation/philosophy.md](../../../foundation/philosophy.md): Abstractions Must Earn Their Place principle (governs the exception policy)
- [docs/foundation/specs.md](../../../foundation/specs.md): spec template this document follows
- `src/app/globals.css`: design token definitions, theme configuration, base layer styles
- `src/lib/utils.ts`: `cn()` utility implementation
- `postcss.config.mjs`: Tailwind 4 PostCSS plugin configuration
- `components.json`: shadcn/ui configuration (baseColor, cssVariables, style)

## Structure

- `README.md`: Tailwind CSS conventions (configuration, theming, dark mode, utility-first policy, class composition, responsive design, exception process)
- `spec/README.md`: this file; reasoning behind the conventions
