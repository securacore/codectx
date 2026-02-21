# Add a go package.
[group('root')]
get package="" *argv="":
  go get {{package}} {{argv}}
