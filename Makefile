#----------------------------------------------------------------------------------
# Base
#----------------------------------------------------------------------------------

ROOTDIR := $(shell pwd)
PACKAGE_PATH:=github.com/solo-io/anyvendor
OUTPUT_DIR ?= $(ROOTDIR)/_output
SOURCES := $(shell find . -name "*.go" | grep -v test.go)
VERSION ?= $(shell git describe --tags)
DEPSGOBIN=$(shell pwd)/.bin

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
	mkdir -p $(DEPSGOBIN)
	PATH=$(DEPSGOBIN):$$PATH go get golang.org/x/tools/cmd/goimports
	PATH=$(DEPSGOBIN):$$PATH go get github.com/golang/protobuf/protoc-gen-go
	PATH=$(DEPSGOBIN):$$PATH go get github.com/envoyproxy/protoc-gen-validate
	PATH=$(DEPSGOBIN):$$PATH go install github.com/envoyproxy/protoc-gen-validate
	PATH=$(DEPSGOBIN):$$PATH go get github.com/golang/mock/gomock
	PATH=$(DEPSGOBIN):$$PATH go install github.com/golang/mock/mockgen
	PATH=$(DEPSGOBIN):$$PATH go install github.com/onsi/ginkgo/ginkgo


.PHONY: mod-download
mod-download:
	go mod download

#----------------------------------------------------------------------------------
# Generated Code
#----------------------------------------------------------------------------------

.PHONY: generated-code
generated-code: $(OUTPUT_DIR)/.generated-code

SUBDIRS:=pkg anyvendor
$(OUTPUT_DIR)/.generated-code:
	PATH=$(DEPSGOBIN):$$PATH mkdir -p ${OUTPUT_DIR}
	PATH=$(DEPSGOBIN):$$PATH $(GO_BUILD_FLAGS) go generate ./...
	PATH=$(DEPSGOBIN):$$PATH goimports -w $(SUBDIRS)
	PATH=$(DEPSGOBIN):$$PATH go mod tidy
	PATH=$(DEPSGOBIN):$$PATH touch $@

# run all tests
# set TEST_PKG to run a specific test package
.PHONY: run-tests
run-tests:
	ginkgo -r -failFast -trace $(GINKGOFLAGS) \
		-ldflags=$(LDFLAGS) \
		-gcflags=$(GCFLAGS) \
		-progress \
		-race \
		-compilers=4 \
		-skipPackage=$(SKIP_PACKAGES) $(TEST_PKG)
