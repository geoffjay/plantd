NOW = "$(shell date +%Y%m%d%H%M%S)"

M := $(shell printf "\033[34;1m▶\033[0m")

db/g/migration: ; $(info $(M) Generating a new migration...)
ifeq ($(MIGRATION),)
	@echo "usage: MIGRATION=create_foo_table make db/g/migration"
else
	@cp logger/templates/db/migration.go logger/db/migrations/$(NOW)_$(MIGRATION).go
endif

.PHONY: up down rollback
