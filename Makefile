BINARY  := cly
PKG     := github.com/pocikode/commitly/internal/version
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
COMMIT  ?= $(shell git rev-parse --short HEAD 2>/dev/null || echo none)
DATE    ?= $(shell date -u +%Y-%m-%dT%H:%M:%SZ)
LDFLAGS := -s -w \
	-X $(PKG).Version=$(VERSION) \
	-X $(PKG).Commit=$(COMMIT) \
	-X $(PKG).Date=$(DATE)

COVER_MIN := 80

.PHONY: build install test vet cover cover-check lint tidy clean smoke

## build: compile a single static binary into bin/oco
build:
	CGO_ENABLED=0 go build -ldflags "$(LDFLAGS)" -o bin/$(BINARY) .

## install: go install with version stamping
install:
	go install -ldflags "$(LDFLAGS)" .

## test: run the test suite
test:
	go test ./...

## vet: run go vet
vet:
	go vet ./...

## cover: run tests with coverage and print the total
cover:
	go test ./... -coverprofile=cover.out
	go tool cover -func=cover.out | tail -1

## cover-check: fail when total coverage drops below COVER_MIN
cover-check: cover
	@total=$$(go tool cover -func=cover.out | grep total: | awk '{print $$3}' | tr -d '%'); \
	echo "total coverage: $$total% (min $(COVER_MIN)%)"; \
	awk "BEGIN{exit !($$total >= $(COVER_MIN))}" || { echo "coverage below $(COVER_MIN)%"; exit 1; }

## tidy: tidy go modules
tidy:
	go mod tidy

## clean: remove build artifacts
clean:
	rm -rf bin dist cover.out

## smoke: build then run --version and --help
smoke: build
	./bin/$(BINARY) --version
	./bin/$(BINARY) --help >/dev/null
