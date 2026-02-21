# Composable Components

Components with sub-components use the dot notation namespace pattern. Sub-components are attached as properties to the main component. For base component conventions, see [components.md](components.md). For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

<rules>

- Sub-components use dot notation: `Card.Header`, `Card.Body`, `Card.Footer`.
- Define the namespace type as an intersection: `type Component = FC<Props> & { SubName: typeof SubName; ... }`.
- Assign sub-components as properties on the parent component after the component definition.
- Set `displayName` on namespaced components.
- The parent component file imports sub-components from the internal `./components` barrel and assembles the namespace.
- Sub-components live in nested directories. Flat files in the parent directory are prohibited.
- Consumers get a single import for the entire component and its sub-components.

</rules>

```typescript
// Card/Card.tsx
import { Body } from "./components/Body";
import { Footer } from "./components/Footer";
import { Header } from "./components/Header";

type Props = {
  children: ReactNode;
};

type Component = FC<Props> & {
  Header: typeof Header;
  Body: typeof Body;
  Footer: typeof Footer;
};

export const Card: Component = ({ children }) => {
  return <div className="card">{children}</div>;
};

Card.displayName = "Card";
Card.Header = Header;
Card.Body = Body;
Card.Footer = Footer;
```

## Usage

```typescript
import { Card } from "@/components/Card";

<Card>
  <Card.Header>Title</Card.Header>
  <Card.Body>Content</Card.Body>
  <Card.Footer>Actions</Card.Footer>
</Card>
```

## Incorrect Structure

```text
Card/
  Card.tsx
  Header.tsx        # WRONG: must be in Header/Header.tsx
  Body.tsx           # WRONG: must be in Body/Body.tsx
  Footer.tsx         # WRONG: must be in Footer/Footer.tsx
```
