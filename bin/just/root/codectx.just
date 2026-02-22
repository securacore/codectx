alias cli := codectx

# Run the built binary.
[group('root')]
codectx *argv="":
  just docker cli {{argv}}
