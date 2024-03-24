PACKAGE = belphegor

MAIN_PATH = cmd/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/
CGO_ENABLED=0

FULL_PATH = $(BUILD_PATH)$(PACKAGE)

VERSION=$(shell git describe --tags --always --abbrev=0 --match='v[0-9]*.[0-9]*.[0-9]*' 2>/dev/null | sed 's/^.//')
COMMIT_HASH=$(shell git rev-parse --short HEAD)
BUILD_TIMESTAMP=$(shell date '+%Y-%m-%dT%H:%M:%S')

FULL_PACKAGE=$(shell go mod edit -json | jq -r '.Module.Path')
LDFLAGS=-ldflags="-X '${FULL_PACKAGE}/internal.Version=${VERSION}' \
                  -X '${FULL_PACKAGE}/internal.CommitHash=${COMMIT_HASH}' \
                  -X '${FULL_PACKAGE}/internal.BuildTime=${BUILD_TIMESTAMP}' \
                  -s -w \
                  -extldflags '-static'"

.phony: run

run:
	go run $(MAIN_PATH) -node_discover=true -debug -scan_delay 1s

build: clean
	go build $(LDFLAGS) -v -o $(BUILD_PATH)$(PACKAGE) $(MAIN_PATH)

build-windows: clean
	GOOS=windows go build $(LDFLAGS) -v -o $(BUILD_PATH)$(PACKAGE).exe $(MAIN_PATH)

install-windows:build-windows
	powershell.exe -command "Copy-Item -Path '$(BUILD_PATH)$(PACKAGE).exe' \
	 -Destination '$(APPDATA)\Microsoft\Windows\Start Menu\Programs\Startup\$(PACKAGE).exe' -Force"


install: build
	sudo cp $(BUILD_PATH)$(PACKAGE) $(INSTALL_PATH)$(PACKAGE)

uninstall:
	sudo rm $(INSTALL_PATH)$(PACKAGE)

clean:
	rm -rf $(FULL_PATH)

clean-windows:
	-powershell.exe -command Remove-Item -Path $(BUILD_PATH)$(PACKAGE).exe -ErrorAction SilentlyContinue

tests:
	go test ./...

lint:
	golangci-lint run

profiling:
	powershell.exe -command Invoke-WebRequest -Uri "http://localhost:8080/debug/pprof/heap" -OutFile "heap.out"
	go tool pprof heap.out

gen-proto:install-proto
	protoc --proto_path=proto --go_out=. proto/*

install-proto:
	@go install google.golang.org/protobuf/cmd/protoc-gen-go@latest