# Invokes AI to audit documentation.
[private]
[group('opencode')]
docs-audit:
  just opencode prompt docs-audit "Run against docs/ directory."

