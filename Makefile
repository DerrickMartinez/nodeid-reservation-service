dep:
		curl https://raw.githubusercontent.com/golang/dep/master/install.sh | sh
		dep ensure -vendor-only

verify: test
		vendor/github.com/kubernetes/repo-infra/verify/verify-boilerplate.sh --rootdir=${CURDIR}
		vendor/github.com/kubernetes/repo-infra/verify/verify-go-src.sh -v --rootdir ${CURDIR}

test:
		go test -v -race $(shell go list ./... | grep -v /vendor/)

BINARY        ?= nodeid-reservation-service
SOURCES        = $(shell find . -name '*.go')
BUILD_FLAGS   ?= -v
LDFLAGS       ?=

build: build/$(BINARY)

build/$(BINARY): $(SOURCES)
		CGO_ENABLED=0 go build -o build/$(BINARY) $(BUILD_FLAGS) -ldflags "$(LDFLAGS)" .

clean:
		@rm -rf build
