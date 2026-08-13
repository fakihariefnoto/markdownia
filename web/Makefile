# Web Makefile — Markdownia embedded frontend
# build feeds Wails' embed step, not a deployable.

.PHONY: dev build preview check clean help

dev: ## Dev server with HMR
	bun run dev

build: ## Build static assets for the Wails embed
	bun run build
	bun run check:no-network

preview: ## Preview the production build
	bun run preview

check: ## Zero-network check on the built bundle
	bun run check:no-network

clean: ## Remove build output
	rm -rf dist node_modules

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
