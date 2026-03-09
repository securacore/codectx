mod docker "bin/just/docker/.mod.just"
mod claude "bin/just/claude/.mod.just"
mod opencode "bin/just/opencode/.mod.just"

import "bin/just/root/.mod.just"

# List available commands.
default:
  just --list --list-submodules
