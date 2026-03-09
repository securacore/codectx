# Invokes AI to generate a commit and then pushes commit.
[private]
[group('ai')]
done:
  just {{ai_bin}} save
  git push
