BIN := stack-pr
BPR := bpr
PKG := ./cmd/stack-pr
BPR_PKG := ./cmd/bpr
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
LDFLAGS := -ldflags "-X github.com/victorhsb/branchless-pr/internal/cli.version=$(VERSION)"

.PHONY: all build test vet fmt fmt-check tidy clean install install-bpr

all: build

build:
	@go build $(LDFLAGS) -o $(BIN) $(PKG)
	@go build $(LDFLAGS) -o $(BPR) $(BPR_PKG)

install:
	@go install $(LDFLAGS) $(PKG)
install-bpr:
	@go install $(LDFLAGS) $(BPR_PKG)
test:
	@go test ./...

vet:
	@go vet ./...

fmt:
	@gofmt -w .

fmt-check:
	@diff="$$(gofmt -l .)"; \
	if [ -n "$$diff" ]; then \
		echo "gofmt issues in:"; echo "$$diff"; exit 1; \
	fi

tidy:
	@go mod tidy

clean:
	@rm -f $(BIN)
