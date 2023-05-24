PROJ_NAME = belphegor

MAIN_PATH = cmd/main.go
BUILD_PATH = build/package/

INSTALL_PATH = /usr/bin/

build: clean
	go build --ldflags '-extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME) $(MAIN_PATH)

build-windows: clean
	GOOS=windows GOARCH=amd64 go build --ldflags '-extldflags "-static"' -v -o $(BUILD_PATH)$(PROJ_NAME).exe $(MAIN_PATH)

install:
	make build
	sudo cp $(BUILD_PATH)$(PROJ_NAME) $(INSTALL_PATH)$(PROJ_NAME)

uninstall:
	sudo rm $(INSTALL_PATH)$(PROJ_NAME)

clean:
	rm -rf $(BUILD_PATH)*a

tests:
	go test ./...

lint:
	golangci-lint run