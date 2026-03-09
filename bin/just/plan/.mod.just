import "../settings.just"

# Invoke a particular plan against the repo.
[group("plan")]
default name="" prompt="":
  just {{codectx_ai_bin}} plan {{name}} {{prompt}}
  exit 0
