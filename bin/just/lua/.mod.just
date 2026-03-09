import "../settings.just"

lua_bin := shell("devbox run -- lua")

# Run a Lua script for the project.
[group("lua")]
default:
  #!/usr/bin/env bash
  echo "HELLO WORLD"
  which lua
  echo "LUA: {{lua_bin}}"
  echo "DONE"
  exit 0
