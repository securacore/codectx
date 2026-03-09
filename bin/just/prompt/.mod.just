import "../settings.just"

# Invoke a particular plan against the repo.
[group("prompt")]
default name="" prompt="":
  just {{codectx_ai_bin}} prompt {{name}} {{prompt}}
  exit 0
