PACKAGE = belphegor

MAIN_PATH = cmd/cli/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/
CGO_ENABLED=0

FULL_PATH = $(BUILD_PATH)$(PACKAGE)

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

# Парсинг текущей версии
CURRENT_VERSION := $(shell git describe --tags --abbrev=0 2>/dev/null || echo "v0.0.0")
MAJOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f1 | tr -d 'v')
MINOR := $(shell echo $(CURRENT_VERSION) | cut -d. -f2)
PATCH := $(shell echo $(CURRENT_VERSION) | cut -d. -f3)


.PHONY: build run install uninstall clean \
    tests lint profiling gen-proto install-proto version tag-patch tag-minor \
    tag-major tag-delete

run:
	go run $(MAIN_PATH) -node_discover=true -debug -scan_delay 1s

build: clean
	go build $(LDFLAGS) -v -o $(BUILD_PATH)$(PACKAGE) $(MAIN_PATH)

install: build
	sudo cp $(BUILD_PATH)$(PACKAGE) $(INSTALL_PATH)$(PACKAGE)

uninstall:
	sudo rm $(INSTALL_PATH)$(PACKAGE)

clean:
	rm -rf $(FULL_PATH)

tests:
	go test ./...

lint:
	golangci-lint run

profiling:
	powershell.exe -command Invoke-WebRequest -Uri "http://localhost:8080/debug/pprof/heap" -OutFile "heap.out"
	go tool pprof heap.out

gen-proto: install-proto
	@protoc --proto_path=proto --go_out=. proto/*

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

tag-patch:
	$(call create_tag,v$(MAJOR).$(MINOR).$$(( $(PATCH) + 1 )))

tag-minor:
	$(call create_tag,v$(MAJOR).$$(( $(MINOR) + 1 )).0)

tag-major:
	$(call create_tag,v$$(( $(MAJOR) + 1 )).0.0)

tag-delete:
	@echo "Current version: $(CURRENT_VERSION)"
	@read -p "Delete tag $(CURRENT_VERSION)? [y/N] " confirm; \
	if [ "$$confirm" = "y" ]; then \
		git tag -d $(CURRENT_VERSION); \
		echo "Tag $(CURRENT_VERSION) deleted"; \
	else \
		echo "Aborted"; \
	fi

version:
	@echo "Version: $(VERSION)"
	@echo "Commit: $(COMMIT_HASH)"
	@echo "Build time: $(BUILD_TIMESTAMP)"

sri-hash:
	@read -p "Enter sha256 hash: " hash; \
	nix hash convert --to sri "sha256:$$hash"

dump:
	@echo "=== START PROJECT CODE DUMP ===" > project_code.txt
	@echo "Created at: $$(date)" >> project_code.txt
	@echo "" >> project_code.txt
	@find . -type f -name "*.go" | while read file; do \
		echo "=== FILE: $$file ===" >> project_code.txt; \
		echo "=== START CODE ===" >> project_code.txt; \
		cat "$$file" >> project_code.txt; \
		echo "=== END CODE ===" >> project_code.txt; \
		echo "" >> project_code.txt; \
	done