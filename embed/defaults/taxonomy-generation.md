# Taxonomy Alias Generation Instructions

You are generating aliases for canonical terms extracted from documentation.
For each canonical term, generate alternative labels that a developer might
use when searching for the same concept.

## Rules

- Generate common abbreviations (e.g., "authentication" -> "auth")
- Generate casual shorthand developers use verbally (e.g., "database" -> "db")
- Generate formal alternatives (e.g., "auth" -> "identity verification")
- Generate plurals and singular forms
- Generate related acronyms (e.g., "JSON Web Token" -> "JWT")
- Do NOT generate antonyms or loosely related concepts
- Do NOT generate more than 10 aliases per term
- Prefer aliases that a developer would actually type into a search query

## Context

Each term is provided with:
- Its parent and child terms in the taxonomy hierarchy
- Example sentences from the source documentation
- The source type (heading, code identifier, or extracted phrase)

Use the hierarchy context to generate aliases that are appropriate to the
term's specificity level. A top-level term like "Authentication" should have
broader aliases than a leaf term like "JWT Refresh Token."
