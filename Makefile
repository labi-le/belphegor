PROJ_NAME = belphegor

MAIN_PATH = cmd/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/
CGO_ENABLED=0

FULL_PATH = $(BUILD_PATH)$(PROJ_NAME)

.phony: run

run:
	go run $(MAIN_PATH) -node_discover=true -debug -scan_delay 1s

build: clean
	go build --ldflags '-s -w -extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME) $(MAIN_PATH)

build-windows: clean
	GOOS=windows go build -ldflags "-s -w -extldflags -static" -v -o $(BUILD_PATH)$(PROJ_NAME).exe $(MAIN_PATH)

install-windows:build-windows
	powershell.exe -command "Copy-Item -Path '$(BUILD_PATH)$(PROJ_NAME).exe' \
	 -Destination '$(APPDATA)\Microsoft\Windows\Start Menu\Programs\Startup\$(PROJ_NAME).exe' -Force"


install: build
	sudo cp $(BUILD_PATH)$(PROJ_NAME) $(INSTALL_PATH)$(PROJ_NAME)

uninstall:
	sudo rm $(INSTALL_PATH)$(PROJ_NAME)

clean:
	rm -rf $(FULL_PATH)

clean-windows:
	-powershell.exe -command Remove-Item -Path $(BUILD_PATH)$(PROJ_NAME).exe -ErrorAction SilentlyContinue

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