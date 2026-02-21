# Docs Audit (AI-Assisted Documentation Review)

This prompt audits the documentation in `docs/` against the full review checklist defined in [docs/foundation/review-standards.md](../../foundation/review-standards.md). It covers formatting compliance, linguistic compliance, example compliance, semantic marker compliance, token positioning, structural integrity, and integration verification, then fixes all issues found. All file references are relative to the repository root.

Read all rules before executing. Execute the steps in the Execution section. Observe the constraints in the Rules section at all times.

## Execution

<execution>

### 1. Gather Documentation

1. Glob all `.md` files in `docs/` recursively, excluding `docs/prompts/`
2. Read every file found. Build a complete picture before analyzing.
3. Read `docs/foundation/specs.md`. This is the spec template that governs all specs.
4. Read `docs/foundation/documentation.md`. This is the authority for documentation structure.
5. Read `docs/foundation/review-standards.md`. This is the complete review checklist. All checks in this audit implement that checklist.
6. Read `docs/foundation/markdown.md`. This is the authority for formatting conventions.
7. Read `docs/foundation/ai-authoring.md`. This is the authority for linguistic patterns and examples.

Do not proceed to analysis until every file has been read.

### 2. Formatting and Linguistic Compliance

For every documentation file (excluding `docs/prompts/`):

1. **Em dash detection.** Search for em dashes (`—`) and double hyphens used as dashes (`--` outside of code blocks and table separators). Flag all occurrences. These are prohibited per `markdown.md`.

2. **Hedging language detection.** Search for hedging words and phrases: "should," "might want to," "consider," "it would be good to," "you may want to." Flag occurrences where these are used in rules or conventions (not in reasoning or spec decisions where qualifying language may be appropriate). Rules are declarative, not suggestions.

3. **Heading depth.** Verify no heading exceeds H3 (`###`). Flag any H4 or deeper.

4. **Positional references.** Search for phrases like "see above," "as mentioned earlier," "the following section," "below." Flag all occurrences. Use explicit section names or links instead.

5. **Consistent terminology.** Note any terms that alternate between synonyms across files (e.g., "config" vs "configuration" vs "settings" for the same concept). Flag inconsistencies.

### 3. Example Compliance

For every documentation file containing `<rules>` sections:

1. Identify every critical rule (rules in `<rules>` blocks that govern behavior).
2. Verify each critical rule has at least one positive example (correct) and one negative example (labeled incorrect).
3. Flag rules that lack negative examples. Simple rules need 1 positive + 1 negative. Complex multi-step rules need 2+ positive + 1+ negative.
4. Verify negative examples are explicitly labeled (e.g., `// Incorrect:` comment, `# WRONG:` marker).
5. Verify examples are realistic, not toy examples.

### 4. Semantic Marker Compliance

For every documentation file (excluding `docs/prompts/`):

1. Identify sections that contain constraints, rules, or behavioral directives.
2. Verify these sections are wrapped in `<rules>` / `</rules>` markers.
3. Verify semantic markers wrap content inside sections, not around headings.
4. Flag constraint sections missing `<rules>` markers.

### 5. Token Positioning

For every documentation file longer than 30 lines:

1. Identify the most critical rules or constraints in the document.
2. Verify these appear in the first 10% of the document (after the H1 and introductory paragraph).
3. Check whether critical constraints are reinforced in the last 10% of the document.
4. Flag documents where critical rules are buried in the middle with no early or late positioning.

### 6. Spec Compliance Check

Find all `spec/README.md` files in `docs/`. For each spec found:

1. **Section compliance.** Verify all sections present are from the allowed set: Purpose, Decisions, Dependencies, Structure. Verify they appear in template order. Flag unknown sections.

2. **Decision format.** Verify each decision entry uses a bold label followed by reasoning. The bold label _names_ the decision as a specific identifier.

3. **Convention restating.** Read the sibling conventions document (the `README.md` in the spec's parent directory, or all convention files if the topic uses a concordance). For each decision entry, check whether the body repeats rules, instructions, examples, file paths, locations, or directives that already appear in the conventions documents.

   A decision entry IS convention restating if it contains:
   - Directives that appear in the conventions doc (e.g., "Use `type` for all definitions" when the conventions doc already says this)
   - Specific file paths, directory locations, or format rules copied from the conventions doc (e.g., "One type per file, PascalCase filename" when the conventions doc already specifies this)
   - Examples identical to those in the conventions doc (e.g., the same `FC.d.ts` example in both files)

   A decision entry is NOT convention restating if it contains:
   - Reasoning for why a choice was made ("Types are more versatile because...")
   - Alternatives that were considered and rejected ("Alternative considered: X, rejected because...")
   - Abstract descriptions of the problem being solved ("Different scopes have conflicting requirements")
   - The decision label itself being specific (e.g., "`type` only, never `interface`" as a label is an identifier, not a restated rule)

4. **Principle alignment.** Verify the spec follows the three principles from the spec template: token density (no padding or ceremony), reasoning over description (why not what), and reproducibility (enough reasoning to recreate the documentation).

### 7. Redundancy Analysis

For every pair of documentation files, check for substantive duplication. Substantive means a full sentence or paragraph that conveys the same information in both files.

The following are NOT redundancy and must be ignored:

- **Entry-point framing.** `docs/README.md` is the documentation entry point. It may restate core facts like the documentation audience ("AI agents first, engineers second") because its job is to immediately set context for every session. This is intentional, not redundant.
- **Scoping statements.** A sentence that declares which conventions apply to which tier (e.g., "Foundational files follow the naming conventions in markdown.md") is a cross-reference with scope. It tells the reader that document X governs context Y. Removing it forces the reader to guess. Do not remove scoping statements.
- **Decision labels.** A spec decision label like "`type` only, never `interface`" names the decision. The conventions doc says "Use `type` for all type definitions. Never use `interface`." These are not duplicates. The label is an identifier; the convention is a rule.

What IS redundancy:

1. **Semantic duplication.** A full sentence or multi-sentence passage that conveys the same information in two files, beyond what is covered by the exclusions above. Quote the overlapping passages with `file:line` references.
2. **Structural duplication.** The same organizational structure (e.g., a list of the same four tiers with the same details) described fully in two files.
3. **Triple statements.** Any fact, claim, or definition that appears in three or more files. These are the highest-priority redundancies.

### 8. Cross-Reference and Integration Integrity

For every `.md` file in `docs/` and for `AGENTS.md` and `CLAUDE.md` at the repository root:

1. Find all relative markdown links (`[text](path)` where path is not a URL and is not inside a backtick code block)
2. Resolve each link relative to the directory containing the source file
3. Verify the target file exists on disk
4. Flag any broken links
5. Verify `(future)` references point to topic directories that do not yet exist. If the directory now exists, the reference is stale and must be replaced with a live link.
6. Verify `docs/README.md` Topics table lists all topic directories that exist in `docs/topics/`.
7. Verify `docs/README.md` Foundational table lists all files in `docs/foundation/` (excluding `spec/`).
8. Verify every spec's Structure section lists all files in its topic directory.
9. Verify every concordance `README.md` table lists all convention files in its directory.

### 9. Report Findings

Output a structured report before making any changes. For each finding:

- **Severity:** High (em dashes, hedging in rules, convention restating in specs, broken links, missing `<rules>` markers), Medium (missing negative examples, semantic duplication across 2 files, stale `(future)` references, heading depth violations), Low (token positioning, positional references, terminology inconsistency, triple statements of low-impact facts)
- **Location:** `file:line` reference for each involved passage
- **Quote:** the exact text that constitutes the finding
- **Proposed fix:** exactly what will be changed, in which file, and why this qualifies as a finding

Group findings by category (Formatting, Linguistic, Examples, Semantic Markers, Token Positioning, Spec Compliance, Redundancy, Cross-References/Integration).

If a potential finding matches any exclusion from the Rules section or the "NOT redundancy" list in Step 3, do not include it in the report.

### 10. Execute Fixes

Apply only the changes listed in the Step 9 report. Do not make changes that are not in the report.

1. **Formatting fixes.** Replace em dashes with appropriate alternatives (commas, colons, semicolons, periods, parentheses). Fix heading depth violations by restructuring or splitting. Remove positional references and replace with explicit section names or links.
2. **Linguistic fixes.** Replace hedging language in rules with declarative statements. "X should be Y" becomes "X is Y" or "Use Y for X."
3. **Example fixes.** Add missing negative examples to critical rules. Label negative examples explicitly.
4. **Semantic marker fixes.** Add `<rules>` / `</rules>` markers to constraint sections that lack them.
5. **Spec compliance fixes.** Rewrite non-compliant decision entries to be reasoning-only. Keep the original decision label unchanged. Remove restated conventions from the body, keeping only reasoning and alternatives rejected.
6. **Redundancy fixes.** When the same content exists in multiple files, keep it in the most authoritative location and remove or replace the duplicate. Authority order: foundational docs > topic conventions > specs > entry point (`docs/README.md`). Replace removed content with a link to the authoritative location if the reader needs access to the information.
7. **Cross-reference and integration fixes.** Fix broken links. Replace stale `(future)` references with live links. Update index tables to reflect actual files. If the target file was moved or renamed, update the link. If the target was deleted, remove the link and any surrounding prose that depends on it.
8. **Verify.** After all fixes, re-run the cross-reference integrity check from Step 8 to confirm no links were broken by the changes.

</execution>

## Rules

<rules>

### What to Fix

- Em dashes and double-hyphen dashes
- Hedging language in rules and conventions (not in spec reasoning)
- Missing negative examples on critical rules
- Missing `<rules>` markers on constraint sections
- Heading depth violations (H4 or deeper)
- Positional references ("see above," "below")
- Specs that restate conventions from the sibling conventions document. Replace restated content with reasoning only, keeping decision labels unchanged.
- Substantive semantic duplication. Full sentences or paragraphs that say the same thing in multiple files.
- Broken cross-reference links
- Stale `(future)` references to directories that now exist
- Index tables that do not match actual files

### What to Never Fix

- **Entry-point framing in `docs/README.md`.** The README may restate core facts (audience, approach) that appear in foundational docs. This is intentional context-setting. Do not remove or shorten these statements.
- **Scoping statements.** Sentences that declare which documents govern which context (e.g., "X follows the conventions in Y") are cross-references with scope, not duplication. Do not remove them.
- **Decision labels in specs.** Labels are identifiers. A label like "`type` only, never `interface`" is specific, not a restated rule. Do not rename, abstract, or shorten decision labels.
- **Index entries in `docs/README.md`.** The Foundational and Topics tables are navigation. Do not remove, reorder, or modify table entries.
- **Content in `docs/prompts/`.** Prompts are not subject to documentation audit. Do not read, analyze, or modify prompt files.

### How to Fix

- Every change must correspond to a specific finding in the Step 9 report. No changes without a cited finding.
- When deduplicating, keep content in the most authoritative location (foundational docs > topic conventions > specs > entry point)
- Do not add, fabricate, or invent decision records, alternatives, or reasoning that were not present in the original content. If a gap exists, note it in the report but do not fill it.
- Do not rewrite prose for style, clarity, or word choice. Only rewrite to fix a specific identified finding.
- All cross-references must resolve after fixes. Verify this as the final step.

### General

- Do not modify files outside `docs/` except `AGENTS.md` and `CLAUDE.md` (for link fixes only)
- Do not create new files
- Do not add AI signatures, co-author lines, or emoji
- If no issues are found, report "No issues found" and exit without making changes

</rules>
