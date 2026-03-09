# Server Actions

Server action conventions and file organization. Server actions are server-side functions invoked from client or server components. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

## Conventions

<rules>

- `"use server"` goes at the top of the file. It marks all exports in the file as server actions.
- One server action per file. This is consistent with the one-export-per-module convention in [docs/topics/typescript/README.md](../typescript/README.md).
- Use arrow functions with named exports. Server actions are not routing files and do not use the `function` keyword exception.
- Use `type Props` for input and `type Return` for output, following the same naming convention as hooks in [docs/topics/react/hooks.md](../react/hooks.md).
- Server actions follow the same colocation pattern as components, hooks, and other organizational directories.

</rules>

## File Organization

Server actions live in an `actions/` directory, following the same scope rules defined in [colocation.md](colocation.md).

<rules>

- `[route]/actions/` for route-specific server actions.
- `[common-ancestor]/actions/` for server actions shared across a route subtree.
- `src/features/[feature]/actions/` for feature-specific server actions.
- Each action lives in its own named directory with its own `index.ts` barrel, consistent with the recursive pattern in [docs/topics/react/file-organization.md](../react/file-organization.md).

</rules>

```text
src/app/(dashboard)/projects/
  actions/
    createProject/
      createProject.ts
      index.ts
    deleteProject/
      deleteProject.ts
      index.ts
    index.ts
```

## Example

```typescript
// src/app/(dashboard)/projects/actions/createProject/createProject.ts
"use server"

type Props = {
  name: string;
  description: string;
};

type Return = {
  success: boolean;
  projectId: string | null;
  error: string | null;
};

export const createProject = async ({ name, description }: Props): Promise<Return> => {
  // validate input
  // call ElysiaJS API or perform server-side operation
  // revalidate path or tag
  // return result
};
```

```typescript
// Consumed in a Server Component
import { createProject } from "./actions/createProject";

export default async function ProjectsPage() {
  return (
    <form action={createProject}>
      <input name="name" />
      <input name="description" />
      <button type="submit">Create</button>
    </form>
  );
}
```

```typescript
// Incorrect: inline "use server" inside a component
export default function ProjectsPage() {
  async function createProject(formData: FormData) {
    "use server"                            // WRONG: mixed concerns, not independently testable
    // ...
  }
  return <form action={createProject}>...</form>;
}

// Incorrect: multiple actions in one file
// src/app/(dashboard)/projects/actions/projectActions.ts
"use server"
export const createProject = async () => { ... };   // WRONG: violates one-export-per-module
export const deleteProject = async () => { ... };   // each action gets its own file

// Incorrect: flat action file without directory
// src/app/(dashboard)/projects/actions/createProject.ts  // WRONG: must be in its own directory
// Correct: src/app/(dashboard)/projects/actions/createProject/createProject.ts
```

## Relationship to ElysiaJS

With ElysiaJS handling the data-layer API, server actions primarily handle form submissions, cache revalidation, and thin orchestration between the frontend and the API. Server actions are not the data access layer. The specific boundary between server actions and ElysiaJS API calls is decided per-feature as the application is built.
