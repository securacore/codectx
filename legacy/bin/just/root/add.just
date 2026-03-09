# Add a devbox CLI package.
[group('root')]
add package="" *argv="":
  devbox add {{package}} {{argv}}
