# Invoke a particular plan against the repo.
[group("root")]
prompt name="" prompt="":
  just {{codectx_ai_bin}} prompt {{name}} {{prompt}}
  exit 0
