# Invoke a particular plan against the repo.
[group("root")]
plan name="" prompt="":
  just {{codectx_ai_bin}} plan {{name}} {{prompt}}
  exit 0
