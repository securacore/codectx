# Bridge Summary Reasoning

Bridge summaries exist to solve a specific problem: when the AI loads
non-adjacent chunks from the same document, it loses the continuity of
what was established between them.

The 30-word limit forces precision. A bridge that tries to summarize
everything becomes a mini-document that defeats the purpose of chunking.
The goal is a pointer — "this is what you'd know if you'd read the
previous chunk" — not a replacement for the content.

Past tense is required because bridges describe what was already covered.
Present tense creates ambiguity about whether the bridge is describing
content the AI should act on or content it should treat as context.

The prohibition on repeating headings prevents lazy bridges that add no
information beyond what the manifest's heading field already provides.
