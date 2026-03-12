# Run the release script.
[group('root')]
release *argv="":
  bash bin/release {{argv}}
