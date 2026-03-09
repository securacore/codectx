# Components

Component definition conventions and props patterns. For composable (namespaced) components, see [composable-components.md](composable-components.md). For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Component Definition

<rules>

- Use arrow function syntax. Do not use the `function` keyword for components.
- Annotate every component with `FC<Props>`. The `FC` type is globally available. Do not import it from React.
- Use named exports. Do not use default exports.
- **Exception:** Next.js routing files (`page.tsx`, `layout.tsx`, etc.) use `function` with default exports as required by the framework.

</rules>

```typescript
// Correct
export const Button: FC<Props> = ({ variant = "primary", children }) => {
  return <button className={variant}>{children}</button>;
};

// Incorrect: function keyword
export function Button({ variant }: Props) { ... }
function Button({ variant }: Props) { ... }

// Incorrect: default export
export default Button;

// Incorrect: React.FC import
import { FC } from "react";
export const Button: React.FC<Props> = ...
```

## Props

<rules>

- Every component declares a `type Props`. This is the input contract.
- Always name the type `Props`. It is module-scoped and never exported, so there is no collision risk.
- `Props` is a standardized name for recognition, not a constraint on shape. The type can be an object, union, primitive, or any other valid TypeScript type.
- Define `Props` as a separate type declaration above the component. Do not use inline props.
- Provide default values for optional props in the destructuring pattern.
- When a prop type is shared across multiple components, promote it to `src/types/` as a domain type.

</rules>

```typescript
// Correct: separate type declaration with defaults in destructuring
type Props = {
  variant?: "primary" | "secondary";
  size?: "sm" | "md" | "lg";
  children: ReactNode;
};

export const Button: FC<Props> = ({
  variant = "primary",
  size = "md",
  children,
}) => { ... };

// Correct: Props can be any shape
type Props = string;
type Props = "primary" | "secondary";
type Props = { items: ReadonlyArray<Item> };

// Incorrect: inline props
export const Button: FC<{ variant: string }> = ...

// Incorrect: props in function params
export const Button = ({ variant }: { variant: string }) => ...

// Incorrect: named after component
type ButtonProps = { variant: string };
```

## Global Types

<rules>

- React types (`FC`, `ReactNode`, `MouseEventHandler`, `HTMLAttributes`, etc.) are globally available via `.d.ts` files in the `types/` directory at the project root. Do not import them from React.
- The `React.*` namespace prefix is prohibited.

</rules>

For the full global types convention, see the Global Types section in [docs/topics/typescript/README.md](../typescript/README.md).
