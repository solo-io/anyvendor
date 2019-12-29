#----------------------------------------------------------------------------------
# Base
#----------------------------------------------------------------------------------

ROOTDIR := $(shell pwd)
PACKAGE_PATH:=github.com/solo-io/protodep
OUTPUT_DIR ?= $(ROOTDIR)/_output
SOURCES := $(shell find . -name "*.go" | grep -v test.go)
VERSION ?= $(shell git describe --tags)

#----------------------------------------------------------------------------------
# Repo init
#----------------------------------------------------------------------------------

# https://www.viget.com/articles/two-ways-to-share-git-hooks-with-your-team/
.PHONY: init
init:
	git config core.hooksPath .githooks

#----------------------------------------------------------------------------------
# Protobufs
#----------------------------------------------------------------------------------

.PHONY: update-deps
update-deps: mod-download
	$(shell cd vendor/github.com/solo-io/protoc-gen-ext; make install)
	GO111MODULE=off go get -u golang.org/x/tools/cmd/goimports
	GO111MODULE=off go get -u github.com/golang/protobuf/protoc-gen-go
	GO111MODULE=off go get -u github.com/envoyproxy/protoc-gen-validate
	GO111MODULE=off go get -u github.com/golang/mock/gomock
	GO111MODULE=off go install github.com/golang/mock/mockgen


.PHONY: mod-download
mod-download:
	go mod download

#----------------------------------------------------------------------------------
# Generated Code
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: $(OUTPUT_DIR)/.generated-code

SUBDIRS:=pkg test
$(OUTPUT_DIR)/.generated-code:
	mkdir -p ${OUTPUT_DIR}
	$(GO_BUILD_FLAGS) go generate ./...
	gofmt -w $(SUBDIRS)
	goimports -w $(SUBDIRS)
	touch $@
#----------------------------------------------------------------------------------
# Cli
#----------------------------------------------------------------------------------

.PHONY: cli
cli: $(OUTPUT_DIR)/protodep

$(OUTPUT_DIR)/protodep:
	go build -o $@ main.go