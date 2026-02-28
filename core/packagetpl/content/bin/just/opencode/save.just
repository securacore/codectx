# Invokes AI to generate a commit.
[private]
[group('opencode')]
save:
  just opencode prompt save "Generate a commit, but do NOT push."
