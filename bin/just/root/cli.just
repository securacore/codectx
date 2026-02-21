# Run the built binary.
[group('root')]
cli *argv="":
  just docker cli {{argv}}
