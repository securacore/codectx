# AI generated commit and pushing changes.
[group('root')]
done:
  echo "Committing and pushing changes, please wait..."
  just prompt save
  git push
  echo "Done"
