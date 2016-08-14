BUILD_PATH          ?= github.com/xiam/hyperfox
BUILD_OUTPUT_DIR    ?= bin
DOCKER_CONTAINER    ?= hyperfox-builder
BUILD_FLAGS					?= -v

GH_ACCESS_TOKEN     ?=
GH_RELEASE_MESSAGE  ?= Latest release.

all: build

build: docker-builder clean
	@mkdir -p $(BUILD_OUTPUT_DIR) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=windows -e GOARCH=amd64 -e CC=x86_64-w64-mingw32-gcc \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_windows_amd64.exe $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=windows -e GOARCH=386 -e CC=i686-w64-mingw32-gcc \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_windows_386.exe $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_linux_amd64 $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=386 \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_linux_386 $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=arm -e GOARM=7 -e CC=arm-linux-gnueabi-gcc \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_linux_armv7 $(BUILD_PATH) && \
	if [[ $$OSTYPE == "darwin"* ]]; then \
		go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/hyperfox_darwin_amd64 $(BUILD_PATH); \
	fi && \
	gzip $(BUILD_OUTPUT_DIR)/hyperfox_linux_* && \
	gzip $(BUILD_OUTPUT_DIR)/hyperfox_darwin_* && \
	zip -r bin/hyperfox_windows_386.zip $(BUILD_OUTPUT_DIR)/hyperfox_windows_386.exe && \
	zip -r bin/hyperfox_windows_amd64.zip $(BUILD_OUTPUT_DIR)/hyperfox_windows_amd64.exe

docker-builder:
	(docker stop $(DOCKER_CONTAINER) || exit 0) && \
	docker build -t $(DOCKER_CONTAINER) .

clean:
	@rm -f *.db && \
	rm -rf $(BUILD_OUTPUT_DIR)
