# Metadata

Metadata conventions for SEO and social sharing. For the reasoning behind these conventions, see [spec/README.md](spec/README.md).

<rules>

- Use the static `metadata` export as the default. Define a `metadata` object in `page.tsx` or `layout.tsx` when the metadata is known at build time.
- Use `generateMetadata` when metadata depends on dynamic data (route params, fetched content, search params).
- Every `page.tsx` and `layout.tsx` that represents a distinct view exports metadata. Every view includes at minimum a `title` and `description` for SEO.
- Metadata inherits from parent layouts. Child routes override only what changes. Do not re-declare inherited metadata.

</rules>

```typescript
// Static metadata (default)
import type { Metadata } from "next";

export const metadata: Metadata = {
  title: "Projects",
  description: "Manage your projects.",
};

export default function ProjectsPage() { ... }
```

```typescript
// Dynamic metadata (when data-dependent)
import type { Metadata } from "next";

type Props = {
  params: Promise<{ projectId: string }>;
};

export const generateMetadata = async ({ params }: Props): Promise<Metadata> => {
  const { projectId } = await params;
  const project = await getProject(projectId);
  return {
    title: project.name,
    description: project.description,
  };
};

export default async function ProjectPage({ params }: Props) { ... }
```
