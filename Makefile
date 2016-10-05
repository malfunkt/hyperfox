SHELL               ?= /bin/bash
BUILD_PATH          ?= github.com/malfunkt/hyperfox
BUILD_OUTPUT_DIR    ?= bin
DOCKER_CONTAINER    ?= hyperfox-builder
BUILD_FLAGS					?= -v
BIN_PREFIX          ?= hyperfox

GH_OWNER            ?= malfunkt
GH_REPO             ?= hyperfox
GH_ACCESS_TOKEN     ?=

build: generate vendor-sync
	go build -o hyperfox github.com/malfunkt/hyperfox

all: docker-build

docker-build: vendor-sync docker-builder clean
	mkdir -p $(BUILD_OUTPUT_DIR) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CC=x86_64-w64-mingw32-gcc \
		-e LD=x86_64-w64-mingw32-ld \
		-e CGO_ENABLED=1 -e GOOS=windows -e GOARCH=amd64 \
		$(DOCKER_CONTAINER) go build -ldflags -H=windowsgui $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_amd64.exe $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CC=i686-w64-mingw32-gcc \
		-e CGO_ENABLED=1 -e GOOS=windows -e GOARCH=386 \
		$(DOCKER_CONTAINER) go build -ldflags -H=windowsgui $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_386.exe $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=amd64 \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_linux_amd64 $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=386 \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_linux_386 $(BUILD_PATH) && \
	docker run \
		-v $$PWD:/app/src/$(BUILD_PATH) \
		-e CC=arm-linux-gnueabi-gcc \
		-e CGO_ENABLED=1 -e GOOS=linux -e GOARCH=arm -e GOARM=7 \
		$(DOCKER_CONTAINER) go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_linux_armv7 $(BUILD_PATH) && \
	if [[ $$OSTYPE == "darwin"* ]]; then \
		GOPARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_darwin_amd64 $(BUILD_PATH) && \
		gzip $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_darwin_*; \
	elif [[ $$OSTYPE == "freebsd"* ]]; then \
		GOARCH=amd64 go build $(BUILD_FLAGS) -o $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_freebsd_amd64 $(BUILD_PATH) && \
		gzip $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_freebsd_*; \
	fi && \
	gzip $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_linux_* && \
	zip -r $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_386.zip $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_386.exe && \
	zip -r $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_amd64.zip $(BUILD_OUTPUT_DIR)/$(BIN_PREFIX)_windows_amd64.exe && \
	rm -f $(BUILD_OUTPUT_DIR)/*.exe

docker-builder:
	(docker stop $(DOCKER_CONTAINER) || exit 0) && \
	docker build -t $(DOCKER_CONTAINER) .

vendor-sync:
	govendor sync

clean:
	rm -f *.db && \
	rm -rf $(BUILD_OUTPUT_DIR)

require-version:
	@if [[ -z "$$VERSION" ]]; then echo "Missing \$$VERSION"; exit 1; fi

require-access-token:
	@if [[ -z "$(GH_ACCESS_TOKEN)" ]]; then echo "Missing \$$GH_ACCESS_TOKEN"; exit 1; fi

release: require-version require-access-token
	RESP=$$(curl --silent --data '{ \
		"tag_name": "v$(VERSION)", \
		"name": "v$(VERSION)", \
		"body": "Release v$(VERSION)", \
		"target_commitish": "$(git rev-parse --abbrev-ref HEAD)", \
		"draft": false, \
		"prerelease": false \
	}' "https://api.github.com/repos/$(GH_OWNER)/$(GH_REPO)/releases?access_token=$(GH_ACCESS_TOKEN)") && \
	\
	UPLOAD_URL_TEMPLATE=$$(echo $$RESP | python -mjson.tool | grep upload_url | awk '{print $$2}' | sed s/,$$//g | sed s/'"'//g) && \
	if [[ -z "$$UPLOAD_URL_TEMPLATE" ]]; then echo $$RESP; exit 1; fi && \
	\
	for ASSET in $$(cd $(BUILD_OUTPUT_DIR) && ls -1 $(BIN_PREFIX)_*); do \
		UPLOAD_URL=$$(echo $$UPLOAD_URL_TEMPLATE | sed s/"{?name,label}"/"?access_token=$(GH_ACCESS_TOKEN)\&name=$$ASSET"/g) && \
		MIME_TYPE=$$(file --mime-type $(BUILD_OUTPUT_DIR)/$$ASSET | awk '{print $$2}') && \
		curl --silent -H "Content-Type: $$MIME_TYPE" --data-binary @bin/$$ASSET $$UPLOAD_URL > /dev/null && \
		echo "-> $$ASSET OK." \
	; done
