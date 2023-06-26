PROJ_NAME = belphegor

MAIN_PATH = cmd/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/

FULL_PATH = $(BUILD_PATH)$(PROJ_NAME)

.phony: run

run:
	go run $(MAIN_PATH) -debug

build: clean
ifeq ($(OS),Windows_NT)
	 $(MAKE) build-windows
else
	go build --ldflags '-extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME) $(MAIN_PATH)
endif

build-windows: clean
	go build --ldflags '-extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME).exe $(MAIN_PATH)

install:
	make build
	sudo cp $(BUILD_PATH)$(PROJ_NAME) $(INSTALL_PATH)$(PROJ_NAME)

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