Version := $(shell git describe --tags --dirty 2> /dev/null)
GitCommit := $(shell git rev-parse HEAD)
LDFLAGS := "-s -w -X github.com/tschaefer/conntrackd/internal/version.Version=$(Version) -X github.com/tschaefer/conntrackd/internal/version.GitCommit=$(GitCommit)"

.PHONY: all
all: fmt lint test dist

.PHONY: fmt
fmt:
	test -z $(shell gofmt -l .) || (echo "[WARN] Fix format issues" && exit 1)

.PHONY: lint
lint:
	test -z $(shell golangci-lint run >/dev/null || echo 1) || (echo "[WARN] Fix lint issues" && exit 1)

.PHONY: dist
dist:
	mkdir -p bin
	CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bin/conntrackd-linux-amd64 -ldflags $(LDFLAGS) .
	CGO_ENABLED=0 GOOS=linux GOARCH=arm64 go build -o bin/conntrackd-linux-arm64 -ldflags $(LDFLAGS) .

.PHONY: checksum
checksum: dist
	cd bin && \
	for f in conntrackd-linux-amd64 conntrackd-linux-arm64; do \
		shasum -a 256 $$f > $$f.sha256; \
	done && \
	cd ..

.PHONY: test
test:
	test -z $(shell go test -v ./internal/... >/dev/null 2>&1 || echo 1) || (echo "[WARN] Fix test issues" && exit 1)

.PHONY: coverage
coverage:
	test -z $(shell go test -coverprofile=coverage.out ./internal/... > /dev/null 2>&1 || echo 1) || (echo "[WARN] Fix coverage issues" && exit 1)
	test -z $(shell go tool cover -html=coverage.out -o coverage.html > /dev/null 2>&1 || echo 1) || (echo "[WARN] Fix coverage issues" && exit 1)

.PHONY: clean
clean:
	rm -rf bin
	rm -f coverage.out coverage.html
