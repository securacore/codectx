# Server and Client Components

Server Component and Client Component conventions for the Next.js App Router. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Server-First Strategy

<rules>

- Every component is a Server Component by default. Do not add `"use client"` unless the component requires it.
- Server Components can fetch data directly, access backend resources, and keep sensitive logic off the client bundle.
- Default to Server Components. Add `"use client"` only when interactivity or browser APIs are needed.

</rules>

## Client Component Boundaries

When a component requires interactivity, extract the smallest possible interactive piece into its own `"use client"` component. The server component composes the client island; it does not become client itself.

<rules>

- `"use client"` goes at the top of the file, always the first line. It marks the boundary: this component and everything it imports become client code.
- Start at the most granular level. Only the interactive piece becomes a client component. Do not default to making the entire module or page client-side.
- Expand the client boundary only when justified. If the entire module genuinely needs to be client-side, that is a conscious decision, not a default.
- Push the `"use client"` boundary as deep into the component tree as possible. A `"use client"` at the page level drags everything underneath into the client bundle.

</rules>

### What Triggers `"use client"`

- Event handlers (`onClick`, `onChange`, `onSubmit`, etc.)
- React hooks that depend on client state or lifecycle (`useState`, `useEffect`, `useRef`, `useReducer`)
- Browser-only APIs (`window`, `document`, `localStorage`, `navigator`)
- Third-party libraries that require a browser environment

### Server Component Composing a Client Island

```typescript
// src/app/(dashboard)/projects/page.tsx
// This is a Server Component (no "use client" directive)

import { ProjectList } from "./components/ProjectList";
import { CreateProjectButton } from "./components/CreateProjectButton";

export default async function ProjectsPage() {
  const projects = await getProjects();     // server-side data fetch
  return (
    <div>
      <h1>Projects</h1>                     {/* server-rendered */}
      <ProjectList projects={projects} />   {/* server-rendered */}
      <CreateProjectButton />               {/* client island */}
    </div>
  );
}
```

```typescript
// src/app/(dashboard)/projects/components/CreateProjectButton/CreateProjectButton.tsx
"use client"

type Props = {};

export const CreateProjectButton: FC<Props> = () => {
  const [open, setOpen] = useState(false);   // requires "use client"
  return <button onClick={() => setOpen(true)}>Create Project</button>;
};
```

```typescript
// Incorrect: entire page marked as client
"use client"    // WRONG: drags all imports into client bundle

export default function ProjectsPage() {
  const projects = await getProjects();   // cannot use async in client components
  // ...
}
```

## Relationship to Error Boundaries

`"use client"` islands embedded in Server Component trees are isolation boundary candidates for `<ErrorBoundary>` wrapping. See [docs/topics/react/error-handling.md](../react/error-handling.md) for the full error handling model.
