# Makefile для unissh.
#
# Главное правило: НИ build, НИ install, НИ release не выполняются без зелёных
# тестов — цели зависят от `test`, поэтому сборка на красных тестах невозможна.
# Собирайте через `make build`, а не голым `go build`.

BINARY    := unissh
PKG       := ./...
VERSION   := $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS   := -s -w -X main.version=$(VERSION)
COVER_MIN ?= 90

.DEFAULT_GOAL := build
.PHONY: all test test-race vet fmt fmt-check lint cover cover-check build install release ci ci-full demo clean

## test: прогнать все тесты (гейт для сборки и релиза)
test:
	go test $(PKG)

## test-race: тесты с детектором гонок (TUI сильно конкурентный)
test-race:
	go test -race $(PKG)

## vet: статический анализ
vet:
	go vet $(PKG)

## fmt: отформатировать код
fmt:
	gofmt -w .

## fmt-check: упасть, если код не отформатирован
fmt-check:
	@out="$$(gofmt -l .)"; if [ -n "$$out" ]; then echo "не отформатировано:"; echo "$$out"; exit 1; fi

## lint: golangci-lint (если установлен)
lint:
	golangci-lint run

## cover: тесты с отчётом покрытия
cover:
	go test -coverprofile=coverage.out $(PKG)
	go tool cover -func=coverage.out | tail -1

## cover-check: гейт покрытия — падает, если ниже COVER_MIN (по умолчанию 90)
cover-check:
	COVER_MIN=$(COVER_MIN) sh scripts/coverage.sh

## build: сборка бинарника — ТОЛЬКО после зелёных тестов, vet и порога покрытия
build: fmt-check vet test cover-check
	go build -ldflags "$(LDFLAGS)" -o $(BINARY) .

## install: установка в $GOBIN — только после зелёных тестов и порога покрытия
install: fmt-check vet test cover-check
	go install -ldflags "$(LDFLAGS)" .

## release: локальный релиз через goreleaser — только после тестов и порога покрытия
release: fmt-check vet test cover-check
	goreleaser release --clean

## ci: то, что гоняет CI (тесты → гонки → vet → покрытие → сборка)
ci: test test-race vet cover-check build

## ci-full: ci + golangci-lint (как оба job'а в GitHub Actions)
ci-full: ci lint

## demo: записать demo.gif на ФИКТИВНЫХ серверах (нужен vhs)
demo:
	./scripts/record-demo.sh

## clean: убрать артефакты
clean:
	rm -f $(BINARY) coverage.out
