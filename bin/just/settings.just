set quiet
set dotenv-load
set dotenv-override
set dotenv-filename := ".env.development"
set shell := ["devbox", "run", "-q", "--"]

# Name of the AI to use. This can either be
# "claude" or "opencode". Changing this changes
# which bin is used to perform actions under the
# bin/just/ai/ module.
codectx_ai_bin := env("CODECTX_AI_BIN", "opencode")
