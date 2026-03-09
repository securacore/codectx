# Bridge Summary Generation Instructions

You are generating one-line semantic bridge summaries for chunk boundaries.
Each bridge summarizes what the previous chunk established that the next
chunk assumes the reader already knows.

## Rules

- Keep each bridge to a single sentence, under 30 words
- Focus on what knowledge carries forward, not what was covered in detail
- Use specific terms from the content, not vague summaries
- Do NOT repeat the heading or title of the previous chunk
- Write in past tense ("Established...", "Defined...", "Covered...")

## Example

Good: "Defined the JWT token structure including header, payload, and
signature fields with RS256 signing requirements."

Bad: "The previous section was about JWT tokens." (too vague)
Bad: "JWT Token Structure: this section covered the structure." (repeats heading)
