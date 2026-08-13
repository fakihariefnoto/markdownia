# Markdownia — root Makefile
#
# Single Go module at the repo root (the Wails app), embedded frontend in web/.
# Packaging configs live in build/. No backend/ or desktop/ projects.

WEB     := web
APP     := markdownia
VERSION := $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
LDFLAGS := -ldflags "-X main.version=$(VERSION) -X github.com/anofac/markdownia/internal/native.AppVersion=$(VERSION)"

.PHONY: build run dev test test-coverage lint security mocks tools \
        web-build web-check package-macos package-windows package-linux \
        release-ci clean help

## -- Build & dev ----------------------------------------------------------

build: ## Build the runnable Wails binary (frontend + Go, embedded)
	$(MAKE) -C $(WEB) build
	wails3 task build

run: ## Run in dev mode
	wails3 task run

dev: ## Dev mode with hot reload
	wails3 task dev

## -- Go quality ------------------------------------------------------------

test: ## Run the Go test suite
	go test ./... -count=1

test-coverage: ## Tests with HTML coverage report
	go test ./... -coverprofile=coverage.out -covermode=atomic
	go tool cover -html=coverage.out -o coverage.html

lint: ## Run golangci-lint (errcheck on — ignored errors skip documents)
	golangci-lint run ./...

security: ## Run gosec + govulncheck
	gosec -exclude=G301,G302,G306 -exclude-dir=build/ios -exclude-dir=build/android ./...
	go run golang.org/x/vuln/cmd/govulncheck@latest ./...

mocks: ## Regenerate mocks (see build/tasks/01-setup.md)
	@echo "mockgen per interface — see build/tasks/01-setup.md"

tools: ## Install dev tools
	go install github.com/go-delve/delve/cmd/dlv@latest
	go install github.com/golangci/golangci-lint/cmd/golangci-lint@latest
	go install github.com/securego/gosec/v2/cmd/gosec@latest

## -- Frontend ---------------------------------------------------------------

web-build: ## Build the frontend
	$(MAKE) -C $(WEB) build

web-check: ## Zero-network check on the frontend bundle
	$(MAKE) -C $(WEB) check

## -- Packaging (per OS; configs in build/) ----------------------------------

package-macos: ## Build .app (universal) + .dmg (requires create-dmg)
	wails3 task darwin:package:universal
	wails3 task darwin:create:dmg

package-windows: ## Build NSIS installer + portable exe (requires NSIS)
	wails3 task windows:package

package-linux: ## Build .deb + .AppImage (requires nfpm + appimagetool)
	wails3 task linux:package

release-ci: ## Full release across all OSes (see .github/workflows/release.yml)
	@echo "Run via the GitHub Actions release workflow (workflow_dispatch only)."

## -- Clean ----------------------------------------------------------------

clean: ## Remove build artifacts
	$(MAKE) -C $(WEB) clean
	rm -rf bin frontend/dist .task coverage.out coverage.html
	rm -rf dist

help: ## Show this help
	@grep -E '^[a-zA-Z_-]+:.*?## .*$$' $(MAKEFILE_LIST) | sort | awk 'BEGIN {FS = ":.*?## "}; {printf "\033[36m%-20s\033[0m %s\n", $$1, $$2}'

.DEFAULT_GOAL := help
