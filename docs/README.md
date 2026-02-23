# Documentation

> [!NOTE]
> Documentation is written for AI agents first and engineers second. For the full explanation of audience and approach, see [documentation.md](foundation/documentation.md). Engineers: the topic directories (`docs/topics/`) contain the conventions for writing code in this project. The foundation documents govern documentation authoring and AI session management. Start with the topics relevant to your work.

All project documentation lives here. This is a manually configured Next.js 16 application using React 19, Tailwind 4, Biome, and Bun. All repository operations run through Just; do not invoke package managers, build tools, or other CLI tools directly. Read the foundational documents before making any decisions. They define how this project operates.

At the start of every session, load [metadata.yml](metadata.yml) and all foundation documents marked `load: always` in the manifest. The metadata manifest maps all documentation entries, their relationships (`depends_on`/`required_by`), and file paths. It is the contextual map for navigating this documentation system. If the task involves writing, editing, reviewing, or restructuring any file under `docs/`, also load all foundation documents marked `load: documentation`. Before making any changes to `metadata.yml`, load [metadata.schema.json](metadata.schema.json) into context. Do not edit the manifest without the schema loaded.

## Foundational

| Document                                              | Purpose                                       |
| ----------------------------------------------------- | --------------------------------------------- |
| [philosophy.md](foundation/philosophy.md)             | Guiding principles for decision-making        |
| [markdown.md](foundation/markdown.md)                 | Markdown formatting conventions               |
| [documentation.md](foundation/documentation.md)       | Documentation management and strategy         |
| [specs.md](foundation/specs.md)                       | Specification template and process            |
| [ai-authoring.md](foundation/ai-authoring.md)         | Cross-model AI authoring conventions          |
| [review-standards.md](foundation/review-standards.md) | Post-update documentation review checklist    |
| [metadata.md](foundation/metadata.md)                 | Metadata manifest conventions and maintenance |

## Product

| Document                                            | Purpose                           |
| --------------------------------------------------- | --------------------------------- |
| [architecture](product/README.md)                   | Product architecture and design   |

## Schemas

| Schema                                                    | Purpose                              |
| --------------------------------------------------------- | ------------------------------------ |
| [codectx.schema.json](schemas/codectx.schema.json)       | Validates `codectx.yml`              |
| [package.schema.json](schemas/package.schema.json)        | Validates `package.yml`              |
| [state.schema.json](schemas/state.schema.json)            | Validates `state.yml` (plan state)   |

## Topics

| Directory                                 | Purpose                                  |
| ----------------------------------------- | ---------------------------------------- |
| [elysiajs](topics/elysiajs/README.md)     | ElysiaJS data-layer API conventions          |
| [go](topics/go/README.md)                 | Go CLI conventions and patterns              |
| [just](topics/just/README.md)             | Command runner conventions and structure     |
| [nextjs](topics/nextjs/README.md)         | Next.js App Router conventions               |
| [react](topics/react/README.md)           | React component conventions and patterns     |
| [tailwind](topics/tailwind/README.md)     | Tailwind CSS 4 conventions and design tokens |
| [typescript](topics/typescript/README.md) | TypeScript conventions and standards         |

## Prompts

| Directory                    | Purpose                                   |
| ---------------------------- | ----------------------------------------- |
| [prompts](prompts/README.md) | AI prompt definitions for automated tasks |

## Maintaining Documentation

- **Adding topic documentation:** Create a subdirectory under `docs/topics/` (lowercase kebab-case) with a `README.md` entry point, then add an entry to the Topics table in this file and a corresponding entry in `metadata.yml`.
- **Adding a foundational document:** Add the file to `docs/foundation/`, then add an entry to the Foundational table in this file and a corresponding entry in `metadata.yml`.
- **Adding a spec:** Create a `spec/` subdirectory within the topic directory with a `README.md` entry point following the template in [specs.md](foundation/specs.md).
- **Adding a prompt:** Create a subdirectory under `docs/prompts/` with a `README.md` entry point, then add an entry to the Prompts table in this file and a corresponding entry in `metadata.yml`.
- **Adding product documentation:** Add the file to `docs/product/`, then add an entry to the Product table in this file and a corresponding entry in `metadata.yml`.
- **After any documentation change:** If the change adds, removes, renames, or restructures documentation files or their relationships, update `docs/metadata.yml` to reflect the change. Maintain `depends_on`/`required_by` symmetry and audit for drift. See [metadata.md](foundation/metadata.md) for the full conventions.
