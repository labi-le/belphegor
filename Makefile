PROJ_NAME = belphegor

MAIN_PATH = cmd/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/
CGO_ENABLED=0

FULL_PATH = $(BUILD_PATH)$(PROJ_NAME)

.phony: run

run:
	go run $(MAIN_PATH) -node_discover=true -debug=false -scan_delay 1s

build: clean
ifeq ($(OS),Windows_NT)
	 $(MAKE) build-windows
else
	go build --ldflags '-s -w -extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME) $(MAIN_PATH)
endif

build-windows: clean
	go build -ldflags "-s -w -extldflags -static" -v -o $(BUILD_PATH)$(PROJ_NAME).exe $(MAIN_PATH)

install:build
ifeq ($(OS),Windows_NT)
	powershell.exe -command "Copy-Item -Path '$(BUILD_PATH)$(PROJ_NAME).exe' \
	 -Destination '$(APPDATA)\Microsoft\Windows\Start Menu\Programs\Startup\$(PROJ_NAME).exe' -Force"
else
	sudo cp $(BUILD_PATH)$(PROJ_NAME) $(INSTALL_PATH)$(PROJ_NAME)
endif

uninstall:
	sudo rm $(INSTALL_PATH)$(PROJ_NAME)

clean:
# if os is windows then remove .exe file
ifeq ($(OS),Windows_NT)
	-powershell.exe -command Remove-Item -Path $(BUILD_PATH)$(PROJ_NAME).exe -ErrorAction SilentlyContinue
else
	rm -rf $(FULL_PATH)
endif

tests:
	go test ./...

lint:
	golangci-lint run

profiling:
	powershell.exe -command Invoke-WebRequest -Uri "http://localhost:8080/debug/pprof/heap" -OutFile "heap.out"
	go tool pprof heap.out

fmt:
	go fmt ./... && betteralign -apply ./...
