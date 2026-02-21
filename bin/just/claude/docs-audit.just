# Invokes AI to audit documentation.
[private]
[group('claude')]
docs-audit:
  just claude prompt docs-audit "Run against docs/ directory."
