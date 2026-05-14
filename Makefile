BIN := stack-pr
PKG := ./cmd/stack-pr
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo "0.1.0")
LDFLAGS := -ldflags "-X github.com/victorhsb/branchless-pr/internal/cli.version=$(VERSION)"

.PHONY: all build test vet fmt fmt-check tidy clean install

all: build

build:
	go build $(LDFLAGS) -o $(BIN) $(PKG)

install:
	go install $(LDFLAGS) $(PKG)

test:
	go test ./...

vet:
	go vet ./...

fmt:
	gofmt -w .

fmt-check:
	@diff="$$(gofmt -l .)"; \
	if [ -n "$$diff" ]; then \
		echo "gofmt issues in:"; echo "$$diff"; exit 1; \
	fi

tidy:
	go mod tidy

clean:
	rm -f $(BIN)
