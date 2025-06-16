app: make dev-app
css: bun run build-css-watch
templ: templ generate --watch -path ./app
broker: make dev-broker
identity: make dev-identity
# Excluded services - use overmind start -x proxy,logger to start without these
# logger: make dev-logger
# proxy: make dev-proxy
state: make dev-state
