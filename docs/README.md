# Documentation

> [!NOTE]
> Documentation is written for AI agents first and engineers second. For the full explanation of audience and approach, see [documentation.md](foundation/documentation/README.md). Engineers: the topic directories (`docs/topics/`) contain the conventions for writing code in this project. The foundation documents govern documentation authoring and AI session management. Start with the topics relevant to your work.

All project documentation lives here. codectx is a Go CLI application. All repository operations run through Just; do not invoke package managers, build tools, or other CLI tools directly. Read the foundational documents before making any decisions. They define how this project operates.

At the start of every session, load [metadata.yml](metadata.yml) and all foundation documents marked `load: always` in the manifest. The metadata manifest maps all documentation entries, their relationships (`depends_on`/`required_by`), and file paths. It is the contextual map for navigating this documentation system. If the task involves writing, editing, reviewing, or restructuring any file under `docs/`, also load all foundation documents marked `load: documentation`. Before making any changes to `metadata.yml`, load [metadata.schema.json](metadata.schema.json) into context. Do not edit the manifest without the schema loaded.

## Foundational

| Document                                              | Purpose                                       |
| ----------------------------------------------------- | --------------------------------------------- |
| [philosophy](foundation/philosophy/README.md)             | Guiding principles for decision-making        |
| [markdown](foundation/markdown/README.md)                 | Markdown formatting conventions               |
| [documentation](foundation/documentation/README.md)       | Documentation management and strategy         |
| [specs](foundation/specs/README.md)                       | Specification template and process            |
| [ai-authoring](foundation/ai-authoring/README.md)         | Cross-model AI authoring conventions          |
| [review-standards](foundation/review-standards/README.md) | Post-update documentation review checklist    |
| [metadata](foundation/metadata/README.md)                 | Metadata manifest conventions and maintenance |

## Product

| Document                                            | Purpose                                             |
| --------------------------------------------------- | --------------------------------------------------- |
| [Architecture](product/README.md)                   | System architecture overview and core concepts       |
| [Package Format](product/packages.md)               | Package structure, manifest, naming, and resolution  |
| [Compilation](product/compilation.md)               | Compile process, content addressing, decomposition   |
| [Compression](product/compression.md)               | CMDX codec, algorithm, tag format, and benchmarks    |
| [Configuration](product/configuration.md)           | Settings, activation, conflicts, and directory layout|
| [Preference Management](product/set-command.md)     | The `codectx set` command and user-local preferences |
| [AI Integration](product/ai-integration.md)         | Entry point linking, loading protocol, watch mode    |
| [Design Decisions](product/spec/README.md)          | Reasoning behind every architectural choice          |

## Schemas

| Schema                                                    | Purpose                              |
| --------------------------------------------------------- | ------------------------------------ |
| [codectx.schema.json](schemas/codectx.schema.json)       | Validates `codectx.yml`              |
| [manifest.schema.json](schemas/manifest.schema.json)        | Validates `manifest.yml`              |
| [plan.schema.json](schemas/plan.schema.json)              | Validates `plan.yml` (plan state)    |

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
- **Adding a foundational document:** Create a subdirectory under `docs/foundation/` (lowercase kebab-case) with a `README.md` entry point, then add an entry to the Foundational table in this file and a corresponding entry in `metadata.yml`.
- **Adding a spec:** Create a `spec/` subdirectory within the topic or foundation directory with a `README.md` entry point following the template in [specs](foundation/specs/README.md).
- **Adding a prompt:** Create a subdirectory under `docs/prompts/` with a `README.md` entry point, then add an entry to the Prompts table in this file and a corresponding entry in `metadata.yml`.
- **Adding product documentation:** Add the file to `docs/product/`, then add an entry to the Product table in this file and a corresponding entry in `metadata.yml`.
- **After any documentation change:** If the change adds, removes, renames, or restructures documentation files or their relationships, update `docs/metadata.yml` to reflect the change. Maintain `depends_on`/`required_by` symmetry and audit for drift. See [metadata](foundation/metadata/README.md) for the full conventions.
