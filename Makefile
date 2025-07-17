PACKAGE = belphegor

MAIN_PATH = cmd/cli/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/
CGO_ENABLED=0

FULL_PATH = $(BUILD_PATH)$(PACKAGE)
PWD = $(shell pwd)

VERSION=$(shell git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null | sed 's/^.//')
COMMIT_HASH=$(shell git rev-parse --short HEAD)
BUILD_TIMESTAMP=$(shell date '+%Y-%m-%dT%H:%M:%S')

FULL_PACKAGE=$(shell go list -m)
METADATA_PACKAGE=${FULL_PACKAGE}/internal
LDFLAGS=-ldflags="-X '${METADATA_PACKAGE}.Version=${VERSION}' \
                  -X '${METADATA_PACKAGE}.CommitHash=${COMMIT_HASH}' \
                  -X '${METADATA_PACKAGE}.BuildTime=${BUILD_TIMESTAMP}' \
                  -s -w \
                  -extldflags '-static'"

CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
MAJOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f1 | tr -d 'v')
MINOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_VERSION) | cut -d. -f3)

.PHONY: run
run:
	go run $(MAIN_PATH) --node_discover=true --debug --scan_delay 1s

.PHONY: build
build: clean
	go build $(LDFLAGS) -v -o $(BUILD_PATH)$(PACKAGE) $(MAIN_PATH)

.PHONY: build-debug
build-debug:
	go build -gcflags="-m=2" $(LDFLAGS) -v -o $(BUILD_PATH)$(PACKAGE) $(MAIN_PATH)

.PHONY: clean
clean:
	rm -rf $(FULL_PATH)

.PHONY: tests
tests:
	go test ./...

.PHONY: lint
lint:
	golangci-lint run

.PHONY: gen-proto
gen-proto: install-proto
	@protoc --proto_path=proto --go_out=. proto/*

.PHONY: install-proto
install-proto:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest

define create_tag
	@echo "Current version: $(CURRENT_VERSION)"
	@NEW_VERSION=$(1); \
	echo "Creating new tag: $$NEW_VERSION"; \
	read -p "Create tag? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		git tag -a $$NEW_VERSION -m "Release $$NEW_VERSION"; \
		echo "Tag $$NEW_VERSION created"; \
	else \
		echo "Aborted"; \
	fi
endef

.PHONY: tag-patch
tag-patch:
	$(call create_tag,v$(MAJOR).$(MINOR).$$(( $(PATCH) + 1 )))

.PHONY: tag-minor
tag-minor:
	$(call create_tag,v$(MAJOR).$$(( $(MINOR) + 1 )).0)

.PHONY: tag-major
tag-major:
	$(call create_tag,v$$(( $(MAJOR) + 1 )).0.0)

.PHONY: tag-delete
tag-delete:
	@echo "Current version: $(CURRENT_VERSION)"
	@read -p "Delete tag $(CURRENT_VERSION)? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		git tag -d $(CURRENT_VERSION); \
		echo "Tag $(CURRENT_VERSION) deleted"; \
	else \
		echo "Aborted"; \
	fi

.PHONY: version
version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build time: $(BUILD_TIMESTAMP)"

.PHONY: sri-hash
sri-hash:
	@read -p "Enter sha256 hash: " hash; \
	nix hash convert --to sri "sha256:$$hash"

.PHONY: dump
dump:
	@echo "=== START PROJECT CODE DUMP ===" > project_code.txt
	@echo "Created at: $$(date)" >> project_code.txt
	@echo "" >> project_code.txt
	@find . -type f \( \
		-name "*.go" -o \
		-name "*.yml" -o \
		-name "*.yaml" -o \
		-name "*.proto" -o \
		-name "*.mod" -o \
		-name "*.sum" -o \
		-name "*.nix" -o \
		-name "Makefile" \
	\) | while read file; do \
		echo "=== FILE: $$file ===" >> project_code.txt; \
		echo "=== START CODE ===" >> project_code.txt; \
		cat "$$file" >> project_code.txt; \
		echo "=== END CODE ===" >> project_code.txt; \
		echo "" >> project_code.txt; \
	done

.PHONY: dist
dist:
	docker run --rm \
        -v $(PWD):/go/src/github.com/labi-le/belphegor \
        -w /go/src/github.com/labi-le/belphegor \
        goreleaser/goreleaser release --clean --snapshot