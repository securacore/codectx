# AI Authoring

This document governs how to write documentation and prompts for cross-model AI consumption. It covers linguistic patterns, instructional structure, and cross-model compatibility. For markdown formatting, see [markdown.md](markdown.md). For documentation organization, see [documentation.md](documentation.md).

**Core principle: write for the floor, not the ceiling.** A capable model always handles well-structured simple input. A less capable model fails on clever or ambiguous input. Clarity is never wasted on a smart model, but it is always missing for a dumb one.

## Baseline Model Assumption

Documentation and prompts target GPT-4o-class models as the baseline for reliable execution. Content must work for models at and below this capability tier without inhibiting frontier models. When in doubt, favor explicitness over brevity.

## Instructional Language

These rules govern how to write instructions, conventions, and rules within documentation and prompts.

- Write in direct imperatives. "Always include a date in YYYY-MM-DD format" not "It would be good to include a date."
- One instruction per sentence. Compound instructions cause partial compliance in mid-tier models.
- Use positive framing. "Always include the header" not "Don't forget to not skip the header."
- Avoid negation chains. State the positive rule, then list exceptions explicitly. "Include the disclaimer. The only exception is [specific case]" not "Never omit the disclaimer unless it's not required."
- Eliminate hedging language. "Each document covers one topic" not "Each document should cover one topic." Rules are declarative, not suggestions.

## Defaults and Exceptions

- State default behavior explicitly. "Use JSON format by default" not "Use the appropriate format."
- Enumerate exceptions as a closed list. End the list with "No other exceptions apply" when the set is exhaustive.
- When rules conflict, provide explicit priority ordering. Lesser models pick arbitrarily without it.

## Definitions and Terminology

- Define terms on first use. Do not assume the model understands project-specific jargon. Example: "The SLA (Service Level Agreement, the contractual uptime commitment to the customer) must be referenced in all incident reports."
- Use the same term for the same concept throughout all documentation. This reinforces the consistent terminology rule in [markdown.md](markdown.md).

## Examples

- Every critical rule includes at least one positive example (correct) and one negative example (labeled incorrect).
- Scale examples to complexity. Simple rules: 1 positive, 1 negative. Complex multi-step rules: 2+ positive, 1+ negative, including edge cases.
- Use realistic examples that mirror actual input complexity. Toy examples teach toy behavior.

## Token Positioning

- Place the most critical rules in the first 10% of the document.
- Repeat the most critical constraints in the last 10% of the document.
- Supporting detail, examples, and edge cases go in the middle.
- This exploits primacy and recency bias present in all transformer-based models.

## Judgment Calls

When a document requires the model to make judgment calls (e.g., "is this redundancy?"), provide explicit classification criteria.

- Define what IS a match using concrete indicators the model can pattern-match against.
- Define what IS NOT a match using concrete exclusions.
- Use lists, not prose, for classification criteria. Lists are parsed more reliably than paragraphs across model tiers.

## Prompt-Specific Patterns

These rules apply specifically to prompt documents in `docs/prompts/`.

- Open with a clear purpose statement in 1-2 sentences.
- Separate execution steps from constraints. Use `<execution>` for steps and `<rules>` for constraints.
- Specify what to omit. Models tend to over-include. Name the things the model must not do.
- Gate sequential dependencies. "Do not proceed to Step 3 until Step 2 is complete."
- End with a verification step. The last step confirms the output is correct.
