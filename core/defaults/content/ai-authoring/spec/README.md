# AI Authoring Specification

Spec for the AI authoring foundation document.

## Purpose

Documentation and prompts must work across AI models of varying capability. Without explicit authoring conventions, content written for frontier models fails when consumed by mid-tier models, and content written for the lowest common denominator fails to leverage capable models. The AI authoring conventions establish linguistic patterns that produce reliable outcomes across the capability spectrum.

## Decisions

- **GPT-4o-class as baseline.** This is the capability floor that most development teams encounter. Content that works at this tier works at all higher tiers. Content that assumes frontier-only capabilities (long-range reasoning, implicit context recovery) fails unpredictably. Alternative considered: targeting the latest frontier model (rejected; excludes teams using mid-tier models and creates fragile content that breaks on model downgrades).

- **Imperative language over descriptive.** "Use X" is more reliably executed than "It would be good to use X." Imperatives reduce ambiguity about whether something is a suggestion or a requirement. Mid-tier models frequently treat hedged language as optional. Alternative considered: softer language for readability (rejected; readability for humans is secondary to execution reliability for AI).

- **One instruction per sentence.** Compound instructions ("Do X and also Y") cause partial compliance in mid-tier models, which process the first clause and drop the second. Single-clause instructions are executed atomically. Alternative considered: allowing compound instructions for conciseness (rejected; conciseness is not worth partial compliance).

- **Token positioning (primacy/recency).** Transformer-based models exhibit attention bias toward the beginning and end of input. Placing critical rules in the first 10% and reinforcing them in the last 10% exploits this bias for better compliance. Alternative considered: organizing purely by logical flow (rejected; logical flow is preserved in the middle section, while compliance-critical rules get positional advantage).

- **`load: documentation` for this document.** AI authoring conventions govern documentation and prompt writing, not code implementation. Loading them in code-only sessions wastes tokens. Alternative considered: `load: always` (rejected; code sessions do not author documentation or prompts).

## Dependencies

- [markdown](../../markdown/README.md): formatting conventions that complement linguistic conventions
- [documentation](../../documentation/README.md): documentation organization referenced for prompt placement
