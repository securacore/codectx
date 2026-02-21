# Install system packages.
[group('root')]
install:
  echo "Installing, please wait..."
  devbox install
  go mod download
  just docker up
  just docker exec go mod tidy
  # echo "Running 'go mod download', please wait..."
  # go mod download
  # echo "Running 'just docker up', please wait..."
  # just docker up
  # echo "Running 'just docker exec go mod tidy', please wait..."
  # just docker exec go mod tidy
  echo "Done"
