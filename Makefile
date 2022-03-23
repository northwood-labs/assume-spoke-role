BINARY_NAME=assume-spoke-role
mkfile_path := $(abspath $(lastword $(MAKEFILE_LIST)))
current_dir := $(dir $(mkfile_path))

#-------------------------------------------------------------------------------
# Global stuff.

# Determine which version of `echo` to use. Use version from coreutils if available.
ECHOCHECK := $(shell command -v /usr/local/opt/coreutils/libexec/gnubin/echo 2> /dev/null)
ifdef ECHOCHECK
    ECHO=/usr/local/opt/coreutils/libexec/gnubin/echo
else
    ECHO=echo
endif

#-------------------------------------------------------------------------------
# Running `make` will show the list of subcommands that will run.

all: help

.PHONY: help
## help: [help]* Prints this help message.
help:
	@ $(ECHO) "Usage:"
	@ $(ECHO) ""
	@ sed -n 's/^##//p' ${MAKEFILE_LIST} | column -t -s ':' | sed -e 's/^/ /' | \
		while IFS= read -r line; do \
			if [[ "$$line" == *"]*"* ]]; then \
				$(ECHO) -e "\033[1;33m$$line\033[0m"; \
			else \
				$(ECHO) "$$line"; \
			fi; \
		done

#-------------------------------------------------------------------------------
# Dependencies

.PHONY: install-tools-goget
## install-tools-goget: [deps] Installs the tools using the Go toolchain.
install-tools-goget:
	go get -u github.com/sqs/goreturns
	go get -u golang.org/x/tools/cmd/goimports
	go get -u mvdan.cc/gofumpt

.PHONY: install-tools-linux
## install-tools-linux: [deps] Installs the tools using Linux-friendly approaches (includes `go get` tools).
install-tools-linux: install-tools-goget
	curl -sfSL https://raw.githubusercontent.com/golangci/golangci-lint/master/install.sh | sh -s -- -b $$(go env GOPATH)/bin

.PHONY: install-tools-mac
## install-tools-mac: [deps]* Installs the tools using Mac-friendly Homebrew (includes `go get` tools).
install-tools-mac: install-tools-goget
	brew install golangci/tap/golangci-lint

#-------------------------------------------------------------------------------
# Clean Go code

.PHONY: clean
## clean: [clean]* Clean Go's module cache.
clean: clean-app clean-go

.PHONY: clean-app
## clean-app: [clean] Cleans local build directories
clean-app:
	rm -Rf ./bin ./dist

.PHONY: clean-go
## clean-go: [clean] clean Go's module cache
clean-go:
	go clean -i -r

.PHONY: clean-go-deep
## clean-go-deep: [clean] deep-clean Go's module cache
clean-go-deep:
	go clean -i -r -x -testcache -modcache -cache

#-------------------------------------------------------------------------------
# Building Go code

.PHONY: build-golang
## build-golang: [build] Compiles the source code into a native binary.
build-golang: godeps
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running go build..."
	mkdir -p ./bin
	go build -ldflags="-s -w -X main.commit=$$(git rev-parse HEAD) -X main.date=$$(date '+%Y-%m-%d') -X main.version=$$(cat ./VERSION | tr -d '\n')" -o bin/$(BINARY_NAME) *.go

.PHONY: build
## build: [build]* Checks dependencies and cleans up go.mod, then compiles the source code.
build: check-deps build-golang

.PHONY: install
## install: [build]* Creates a hard-link of the compiled binary into `/usr/local/bin`.
install:
	cp -vf -l bin/$(BINARY_NAME) /usr/local/bin/$(BINARY_NAME)

#-------------------------------------------------------------------------------
# Golang docs

.PHONY: docs
docs:
	@ MODPATH="$$(cat go.mod | grep ^module | awk '{print $$2}')" && echo "http://localhost:6060/pkg/$$MODPATH" && open "http://localhost:6060/pkg/$$MODPATH" && godoc -http=:6060

#-------------------------------------------------------------------------------
# Testing

.PHONY: test
## test: [testing] Run ALL testing tasks.
test:
	@ $(ECHO) "no-op"

#-------------------------------------------------------------------------------
# Linting

.PHONY: check-deps
## check-deps: [lint] Checks that the appropriate dependencies are installed with the right versions.
check-deps:
	@ $(ECHO) " "
	@ $(ECHO) "=====> Checking dependencies..."
	@ scripts/dependency-pip.py
	@ scripts/dependency-check.py

.PHONY: godeps
## godeps: [lint] go.mod lint, update, and download.
godeps:
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running go get -v..."
	go mod tidy -go=1.17 && go get -v ./...

.PHONY: gofmt
## gofmt: [lint] Rewrite the source files to match canonical Go format.
gofmt:
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running gofumpt and goimports..."
	gofumpt -s -w . && goimports -e -w .

.PHONY: golint
## golint: [lint] Lints all Golang code.
golint: godeps gofmt
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running golangci-lint..."
	golangci-lint run --fix *.go

.PHONY: pylint
## pylint: [lint] Lints all Python code.
pylint:
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running yapf..."
	find . -type f -name "*.py"  -not -path "*/venv/*" | xargs -I% yapf --in-place "%"

.PHONY: markdownlint
## markdownlint: [lint] Lints all Markdown files.
markdownlint:
	@ $(ECHO) " "
	@ $(ECHO) "=====> Running Markdownlint..."
	npx markdownlint-cli --fix '**/*.md' --ignore 'node_modules'

.PHONY: lint
## lint: [lint]* Runs ALL linting tasks.
lint: check-deps markdownlint pylint gofmt golint

#-------------------------------------------------------------------------------
# Git Tasks

.PHONY: tag
## tag: [release]* Git tags (and GPG-signs) the release.
tag:
	@ if [ $$(git status -s -uall | wc -l) != 1 ]; then echo 'ERROR: Git workspace must be clean.'; exit 1; fi;

	@echo "This release will be tagged as: $$(cat ./VERSION)"
	@echo "This version should match your release. If it doesn't, re-run 'make version'."
	@echo "---------------------------------------------------------------------"
	@read -p "Press any key to continue, or press Control+C to cancel. " x;

	@echo " "
	@chag update $$(cat ./VERSION)
	@echo " "

	@echo "These are the contents of the CHANGELOG for this release. Are these correct?"
	@echo "---------------------------------------------------------------------"
	@chag contents
	@echo "---------------------------------------------------------------------"
	@echo "Are these release notes correct? If not, cancel and update CHANGELOG.md."
	@read -p "Press any key to continue, or press Control+C to cancel. " x;

	@echo " "

	git add .
	git commit -a -m "Preparing the $$(cat ./VERSION) release."
	chag tag --sign

.PHONY: version
## version: [release]* Sets the version for the next release. Pre-requisite for a release tag.
version:
	@echo "Current version: $$(cat ./VERSION)"
	@read -p "Enter new version number: " nv; \
	printf "$$nv" > ./VERSION
