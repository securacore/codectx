# Taxonomy Generation Reasoning

The alias generation instructions prioritize search recall — making sure
developers find the right documentation regardless of which synonym they use.

The 10-alias limit prevents taxonomy bloat. Early testing showed that beyond
10 aliases per term, the additional aliases tend to be low-quality and can
cause false positive matches in BM25 search.

The prohibition on antonyms and loose associations prevents the taxonomy
from creating misleading connections. "Error handling" should not alias to
"success response" even though they're conceptually related — a developer
searching for error handling does not want success response documentation.
