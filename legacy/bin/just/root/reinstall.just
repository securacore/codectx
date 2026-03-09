# Re-install system packages.
[group('root')]
reinstall:
  just clean --destroy
  just install
