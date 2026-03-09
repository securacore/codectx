# AI Authoring Specification

Spec for the AI authoring foundation document.

## Purpose

Documentation and prompts must work across AI models of varying capability. Without explicit authoring conventions, content written for frontier models fails when consumed by mid-tier models, and content written for the lowest common denominator fails to leverage capable models. The AI authoring conventions establish linguistic patterns that produce reliable outcomes across the capability spectrum.

## Decisions

- **Configurable model class via `ai.class` preference.** The baseline model class is read from `.codectx/preferences.yml` at `ai.class` rather than hardcoded into the document. This allows the same authoring conventions to adapt to the project's chosen capability floor. The document uses a meta-reference pattern: it tells the AI to check the preference and adjust its behavior accordingly. Alternative considered: hardcoding a specific model class (rejected; prevents projects from targeting different capability tiers without forking the document). Alternative considered: separate documents per model class (rejected; creates maintenance burden and content drift between variants).

- **Known model classes as a validated set.** The `ai.class` preference accepts only known class identifiers (`gpt-4o-class`, `claude-sonnet-class`, `o1-class`). This ensures documentation compatibility targets are meaningful and well-defined. The default for new projects is `gpt-4o-class`, the most conservative baseline. Alternative considered: free-form string (rejected; arbitrary values have no semantic meaning and cannot guide authoring behavior). Alternative considered: abstract tiers like "low/medium/high" (rejected; abstract tiers are ambiguous and do not map to concrete model behaviors).

- **Imperative language over descriptive.** "Use X" is more reliably executed than "It would be good to use X." Imperatives reduce ambiguity about whether something is a suggestion or a requirement. Mid-tier models frequently treat hedged language as optional. Alternative considered: softer language for readability (rejected; readability for humans is secondary to execution reliability for AI).

- **One instruction per sentence.** Compound instructions ("Do X and also Y") cause partial compliance in mid-tier models, which process the first clause and drop the second. Single-clause instructions are executed atomically. Alternative considered: allowing compound instructions for conciseness (rejected; conciseness is not worth partial compliance).

- **Token positioning (primacy/recency).** Transformer-based models exhibit attention bias toward the beginning and end of input. Placing critical rules in the first 10% and reinforcing them in the last 10% exploits this bias for better compliance. Alternative considered: organizing purely by logical flow (rejected; logical flow is preserved in the middle section, while compliance-critical rules get positional advantage).

- **`load: documentation` for this document.** AI authoring conventions govern documentation and prompt writing, not code implementation. Loading them in code-only sessions wastes tokens. Alternative considered: `load: always` (rejected; code sessions do not author documentation or prompts).

## Dependencies

- [markdown](../../markdown/README.md): formatting conventions that complement linguistic conventions
- [documentation](../../documentation/README.md): documentation organization referenced for prompt placement
