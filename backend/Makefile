.DEFAULT_GOAL := serve

GO ?= go
ADDR ?= 127.0.0.1:8080
CONFIG ?=
export OPENCV_PREFIX

.PHONY: build serve fmt test check docker-test

build:
	@mkdir -p bin
	./scripts/with-opencv.sh $(GO) build -mod=readonly -o bin/checkbox-server .

serve: build
	./bin/checkbox-server -addr "$(ADDR)" -config "$(CONFIG)"

fmt:
	gofmt -w *.go internal

test:
	./scripts/with-opencv.sh $(GO) test -mod=readonly -tags matprofile ./...

check:
	@test -z "$$(gofmt -l *.go internal)" || { printf 'Run make fmt\n'; exit 1; }
	$(GO) mod verify
	./scripts/with-opencv.sh $(GO) vet -mod=readonly ./...
	$(MAKE) test
	$(MAKE) build

docker-test:
	docker build -t checkbox-detector:test .
	docker run --rm --entrypoint go checkbox-detector:test test -mod=readonly -tags matprofile ./...
