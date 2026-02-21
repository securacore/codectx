# Tailwind

Tailwind CSS 4 conventions for this repository. Tailwind 4 uses CSS-first configuration: there is no JavaScript config file. All configuration lives in `src/app/globals.css` via native CSS directives (`@theme`, `@custom-variant`, `@import`). The design token system uses CSS custom properties in the oklch color space.

For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Configuration

<rules>

- `postcss.config.mjs` uses the `@tailwindcss/postcss` plugin. This is the Tailwind 4 PostCSS integration; do not use the legacy `tailwindcss` plugin.
- `src/app/globals.css` is the single configuration source. All theme tokens, custom variants, and base layer styles are defined here.
- There is no `tailwind.config.ts` or `tailwind.config.js`. Do not create one. Tailwind 4 replaces the JavaScript config with CSS-native directives.
- `@import "tailwindcss"` replaces the legacy `@tailwind base`, `@tailwind components`, and `@tailwind utilities` directives.

</rules>

```javascript
// Correct: Tailwind 4 PostCSS integration in postcss.config.mjs
export default {
  plugins: {
    "@tailwindcss/postcss": {},
  },
};
```

```css
/* Correct: Tailwind 4 CSS entry point (globals.css) */
@import "tailwindcss";
@import "tw-animate-css";

@custom-variant dark (&:is(.dark *));

@theme inline {
  /* token mappings */
}
```

```css
/* Incorrect: legacy Tailwind 3 directives */
@tailwind base;       /* WRONG: use @import "tailwindcss" */
@tailwind components; /* WRONG: use @import "tailwindcss" */
@tailwind utilities;  /* WRONG: use @import "tailwindcss" */
```

```javascript
/* Incorrect: legacy PostCSS plugin */
export default {
  plugins: {
    tailwindcss: {},  /* WRONG: use @tailwindcss/postcss */
  },
};
```

## Theming

Design tokens are defined as CSS custom properties and registered with Tailwind via the `@theme inline` directive. This makes them available as utility classes (e.g., `bg-primary`, `text-muted-foreground`, `rounded-lg`).

<rules>

- Design tokens are defined in `:root` (light theme) and `.dark` (dark theme override) blocks in `globals.css`.
- All color tokens use the `oklch()` color space. oklch is perceptually uniform, making it suitable for programmatic color manipulation and consistent contrast ratios.
- The `@theme inline` block maps CSS custom properties to Tailwind token names. The `inline` keyword means these values reference the `:root`/`.dark` variables rather than emitting standalone custom properties.
- Token naming follows the pattern established in `globals.css`: `--background`, `--foreground`, `--primary`, `--primary-foreground`, `--muted`, `--muted-foreground`, etc. Each semantic role has a base and a `-foreground` variant.

</rules>

```css
/* Adding a new design token: */

/* 1. Define the CSS custom property in :root and .dark */
:root {
  --success: oklch(0.65 0.15 145);
  --success-foreground: oklch(0.98 0 0);
}
.dark {
  --success: oklch(0.45 0.12 145);
  --success-foreground: oklch(0.98 0 0);
}

/* 2. Register it in @theme inline */
@theme inline {
  --color-success: var(--success);
  --color-success-foreground: var(--success-foreground);
}

/* Now bg-success, text-success-foreground, etc. are available as utilities. */
```

```css
/* Incorrect: hardcoded color in @theme without CSS custom property */
@theme inline {
  --color-success: oklch(0.65 0.15 145);  /* WRONG: no dark mode support */
}

/* Incorrect: hex or rgb color values */
:root {
  --success: #22c55e;  /* WRONG: use oklch() */
}
```

## Dark Mode

<rules>

- Dark mode is class-based, configured via `@custom-variant dark (&:is(.dark *))` in `globals.css`.
- The `.dark` class on an ancestor element activates dark mode for its subtree. This enables runtime theme switching without media query dependency.
- Dark mode overrides are defined in the `.dark` block in `globals.css`, using the same CSS custom property names as the `:root` block with adjusted values.
- Use Tailwind's `dark:` variant for component-level dark mode adjustments when the design token system does not cover the need. Add a design token instead of using `dark:` on individual elements when possible.

</rules>

```css
/* Correct: class-based dark mode via CSS custom property override */
:root {
  --card: oklch(0.98 0 0);
}
.dark {
  --card: oklch(0.2 0 0);
}
/* bg-card automatically resolves to the correct value based on .dark class presence. */
```

```css
/* Incorrect: media query dark mode */
@media (prefers-color-scheme: dark) {
  :root {
    --card: oklch(0.2 0 0);  /* WRONG: bypasses class-based switching, couples to OS preference */
  }
}
```

## Utility-First Policy

Tailwind utilities are the default for all styling. Custom CSS is a last resort, not a convenience. This convention is an application of the [Abstractions Must Earn Their Place](../../foundation/philosophy.md) principle: every deviation from the utility system must demonstrably justify its existence.

<rules>

- Use Tailwind utility classes for all styling. This is the default, not a preference.
- No custom CSS classes unless the utility approach is demonstrably more burdensome than a custom class would be. "More burdensome" means the utility composition is significantly harder to read, maintain, or reason about, not merely longer.
- `@apply` is restricted to the `@layer base` block in `globals.css` for global base styles. Do not use `@apply` in component styles or in any other context.
- Arbitrary values (`bg-[#ff0000]`, `p-[13px]`, `grid-cols-[1fr_2px_1fr]`) are permitted only when absolutely necessary for one-off situations where no design token, Tailwind utility, or Tailwind package covers the need.
- Prefer Tailwind ecosystem packages for extending the utility set, following the [Leverage Before Building](../../foundation/philosophy.md) principle. Evaluate third-party packages critically: maintenance activity, community adoption, bundle size, API quality, and security posture. Each package adds to the design surface area and the supply chain.

</rules>

```typescript
// Correct: Tailwind utilities
<div className="flex items-center gap-4 rounded-lg bg-card p-6 text-card-foreground" />

// Correct: conditional classes via cn()
<div className={cn("flex items-center gap-4", isActive && "bg-primary text-primary-foreground")} />

// Incorrect: custom CSS class
<div className="card-container" />  // WRONG: use Tailwind utilities

// Incorrect: @apply in a component
// styles.module.css
// .card { @apply flex items-center gap-4; }  // WRONG: @apply only in globals.css base layer
```

### Design System Exceptions

Exceptions to the utility-first policy (custom CSS classes, arbitrary values, `@apply` outside the base layer) require rigorous scrutiny before implementation.

<rules>

- The AI's default posture on design system exceptions is skepticism, not neutrality. When an exception is proposed, the AI actively searches for alternatives: existing design tokens, Tailwind utilities, Tailwind packages, `@theme` extensions.
- The AI walks the full alternative path before considering an exception valid: "Can an existing token solve this? Can we add a token? Can a Tailwind package handle this? Is there a utility composition that works?"
- Both the engineer and the AI must agree on the reasoning before implementation. The AI pushes back harder than a peer reviewer, specifically to protect less experienced engineers from eroding the design system.
- Every approved exception requires a colocated code comment documenting the exception and its justification. The comment explains why the design system could not accommodate the need.
- Exception volume is a diagnostic signal. A growing number of exception comments across the codebase indicates fundamental gaps in the design system, not a need for more exceptions. Address the root cause.

</rules>

```typescript
// Correct: approved exception with documentation
<div
  className="grid-cols-[1fr_2px_1fr]"  // Exception: separator requires a 2px column that has no design token equivalent.
/>

// Incorrect: arbitrary value without documentation
<div className="mt-[7px]" />  // WRONG: no documented justification. Use a spacing utility or add a token.

// Incorrect: exception without scrutiny
<div className="bg-[#1a2b3c]" />  // WRONG: use a design token. If the color is needed, add it to the theme.
```

## Class Composition

<rules>

- Use `cn()` (from `src/lib/utils.ts`) for all conditional class composition. `cn()` combines `clsx` (conditional class building) with `tailwind-merge` (intelligent class deduplication). Do not use manual template literals or string concatenation for conditional classes.
- Use `cva()` (from `class-variance-authority`) for component variant APIs. `cva()` defines a base set of classes and named variants with a type-safe API. Use it when a component has distinct visual variants controlled by props.
- `cn()` and `cva()` are project standards, not optional utilities. Every component that has conditional or variant-driven classes uses one or both.

</rules>

```typescript
import { cn } from "@/lib/utils";
import { cva, type VariantProps } from "class-variance-authority";

// cn() for conditional classes
type Props = {
  isActive?: boolean;
  className?: string;
};

export const NavItem: FC<Props> = ({ isActive, className }) => (
  <a className={cn("px-3 py-2 text-sm rounded-md", isActive && "bg-accent text-accent-foreground", className)}>
    {/* ... */}
  </a>
);

// cva() for component variant APIs
const buttonVariants = cva("inline-flex items-center justify-center rounded-md text-sm font-medium", {
  variants: {
    variant: {
      default: "bg-primary text-primary-foreground",
      secondary: "bg-secondary text-secondary-foreground",
      destructive: "bg-destructive text-destructive-foreground",
      outline: "border border-input bg-background",
      ghost: "hover:bg-accent hover:text-accent-foreground",
    },
    size: {
      default: "h-10 px-4 py-2",
      sm: "h-9 px-3",
      lg: "h-11 px-8",
    },
  },
  defaultVariants: {
    variant: "default",
    size: "default",
  },
});

type Props = VariantProps<typeof buttonVariants> & {
  className?: string;
  children: ReactNode;
};

export const Button: FC<Props> = ({ variant, size, className, children }) => (
  <button className={cn(buttonVariants({ variant, size }), className)}>
    {children}
  </button>
);
```

```typescript
// Incorrect: manual template literals
<div className={`px-3 py-2 ${isActive ? "bg-accent" : ""}`} />  // WRONG: use cn()

// Incorrect: string concatenation
<div className={"px-3 py-2 " + (isActive ? "bg-accent" : "")} />  // WRONG: use cn()
```

## Responsive Design

<rules>

- Mobile-first. Build for mobile viewports, then layer styles for larger viewports using Tailwind's `min-width` breakpoint utilities (`sm:`, `md:`, `lg:`, `xl:`, `2xl:`).
- Use Tailwind's default breakpoint system. Do not define custom breakpoints unless the application's layout requirements genuinely demand them (this would be a design system exception requiring the scrutiny process described above).
- Responsive utilities are applied inline alongside other utilities. Do not create separate mobile/desktop component variants unless the layouts are fundamentally different in structure, not just styling.

</rules>

```typescript
// Correct: mobile-first responsive
<div className="flex flex-col gap-2 md:flex-row md:gap-4 lg:gap-6" />

// Correct: responsive text sizing
<h1 className="text-2xl font-bold md:text-3xl lg:text-4xl" />

// Incorrect: desktop-first (using max-width thinking)
<div className="flex-row max-md:flex-col" />  // WRONG: start with mobile layout, add breakpoints up
```

## Key Constraints

<rules>

- No `tailwind.config.ts`. All configuration in `src/app/globals.css`.
- All colors in `oklch()`. No hex, rgb, or hsl.
- Utility classes are the default. Custom CSS, arbitrary values, and `@apply` outside the base layer require the exception process.
- Every approved exception has a colocated code comment with justification.
- `cn()` for conditional classes. `cva()` for variant APIs. No manual string concatenation.
- Mobile-first responsive design. Build for mobile, layer up with `min-width` breakpoints.

</rules>
