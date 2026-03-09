# AI Authoring

How to write documentation and prompts for cross-model AI consumption. This covers linguistic patterns, instructional structure, and cross-model compatibility. For markdown formatting, see [markdown](../markdown/README.md). For documentation organization, see [documentation](../documentation/README.md).

**Core principle: write for the floor, not the ceiling.** A capable model always handles well-structured simple input. A less capable model fails on clever or ambiguous input. Clarity is never wasted on a smart model, but it is always missing for a dumb one.

## Baseline Model Class

The target model class for this project is configured in `.codectx/preferences.yml` under the `ai.class` key. Check that file to determine the baseline. If `ai.class` is not set, default to `gpt-4o-class` behavior.

Adapt documentation style to the configured class:

- **`gpt-4o-class`** — Mid-tier instruction-following models. Write with maximum explicitness. One instruction per sentence. Enumerate all exceptions. Avoid implicit reasoning chains. This is the most conservative baseline.
- **`claude-sonnet-class`** — Strong reasoning models with extended context. Moderate explicitness is sufficient. Compound instructions are acceptable when logically grouped. Implicit context recovery is reliable for adjacent content.
- **`o1-class`** — Frontier reasoning models. Dense, precise language is preferred over redundant explicitness. Multi-step reasoning chains work without decomposition. Focus on correctness and completeness over hand-holding.

Regardless of class, all documentation must be structurally sound: clear headings, defined terms, explicit defaults, and concrete examples. The class setting adjusts linguistic density, not structural rigor.

## Instructional Language

<rules>

- Write in direct imperatives. "Always include a date in YYYY-MM-DD format" not "It would be good to include a date."
- One instruction per sentence. Compound instructions cause partial compliance in mid-tier models.
- Use positive framing. "Always include the header" not "Don't forget to not skip the header."
- Avoid negation chains. State the positive rule, then list exceptions explicitly. "Include the disclaimer. The only exception is [specific case]" not "Never omit the disclaimer unless it's not required."
- Eliminate hedging language. "Each document covers one topic" not "Each document should cover one topic." Rules are declarative, not suggestions.

</rules>

## Defaults and Exceptions

<rules>

- State default behavior explicitly. "Use JSON format by default" not "Use the appropriate format."
- Enumerate exceptions as a closed list. End the list with "No other exceptions apply" when the set is exhaustive.
- When rules conflict, provide explicit priority ordering. Lesser models pick arbitrarily without it.

</rules>

## Definitions and Terminology

<rules>

- Define terms on first use. Do not assume the model understands project-specific jargon.
- Use the same term for the same concept throughout all documentation. This reinforces the consistent terminology rule in [markdown](../markdown/README.md).

</rules>

## Examples

<rules>

- Every critical rule includes at least one positive example (correct) and one negative example (labeled incorrect).
- Scale examples to complexity. Simple rules: 1 positive, 1 negative. Complex multi-step rules: 2+ positive, 1+ negative, including edge cases.
- Use realistic examples that mirror actual input complexity. Toy examples teach toy behavior.

</rules>

## Token Positioning

<rules>

- Place the most critical rules in the first 10% of the document.
- Repeat the most critical constraints in the last 10% of the document.
- Supporting detail, examples, and edge cases go in the middle.
- This exploits primacy and recency bias present in all transformer-based models.

</rules>

## Judgment Calls

When a document requires the model to make judgment calls, provide explicit classification criteria.

<rules>

- Define what IS a match using concrete indicators the model can pattern-match against.
- Define what IS NOT a match using concrete exclusions.
- Use lists, not prose, for classification criteria. Lists are parsed more reliably than paragraphs across model tiers.

</rules>

## Prompt-Specific Patterns

These rules apply specifically to prompt documents in `docs/prompts/`.

<rules>

- Open with a clear purpose statement in 1-2 sentences.
- Separate execution steps from constraints. Use `<execution>` for steps and `<rules>` for constraints.
- Specify what to omit. Models tend to over-include. Name the things the model must not do.
- Gate sequential dependencies. "Do not proceed to Step 3 until Step 2 is complete."
- End with a verification step. The last step confirms the output is correct.

</rules>
