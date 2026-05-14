BIN := stack-pr
PKG := ./cmd/stack-pr

.PHONY: all build test vet fmt fmt-check tidy clean install

all: build

build:
	go build -o $(BIN) $(PKG)

install:
	go install $(PKG)

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
