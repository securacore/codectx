# Philosophy Specification

Spec for the philosophy foundation document.

## Purpose

Every project needs a shared decision-making framework that AI tools and engineers can reference when choosing between competing approaches. Without explicit principles, each session makes independent judgment calls that drift over time. The philosophy document codifies the baseline heuristics so decisions are consistent.

## Decisions

- **Generic principles, not prescriptive methodology.** The philosophy covers universal software development heuristics (consistency, clarity, abstraction discipline) rather than prescribing specific methodologies like SOLID or TDD. Projects vary in their technical approach; the embedded defaults provide a starting framework that users customize. Alternative considered: including SOLID/DRY explicitly (rejected; too opinionated for a default that applies to all project types).

- **`load: always` for philosophy.** This is the only default document loaded every session. AI tools reference these principles when making implementation decisions, not just when writing documentation. Loading it always ensures the decision-making framework is present regardless of task type. Alternative considered: `load: documentation` (rejected; philosophy informs code decisions, not just doc decisions).

- **Principle ordering by frequency of use.** Consistency and clarity are the most commonly invoked principles during development, so they appear first. More specialized principles (leverage before building, configuration is truth) appear later.

## Dependencies

- [ai-authoring](../../ai-authoring/README.md): referenced by the "Write for the Floor" principle
