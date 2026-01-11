BINARY := unifi-dns-sync
PKG := ./...
CMD := ./cmd
DIST := dist

.PHONY: help
help:
	@printf "%s\n" "Targets:" \
		"  make build        Build local binary" \
		"  make run          Run from source" \
		"  make test         Run tests" \
		"  make fmt          Go fmt" \
		"  make vet          Go vet" \
		"  make tidy         Go mod tidy" \
		"  make lint         Fmt + vet" \
		"  make clean        Remove build artifacts" \
		"  make dist         Cross-compile into ./$(DIST)"

.PHONY: build
build:
	@mkdir -p bin
	go build -o bin/$(BINARY) $(CMD)

.PHONY: run
run:
	go run $(CMD)

.PHONY: test
test:
	go test $(PKG)

.PHONY: fmt
fmt:
	go fmt $(PKG)

.PHONY: vet
vet:
	go vet $(PKG)

.PHONY: tidy
tidy:
	go mod tidy

.PHONY: lint
lint: fmt vet

.PHONY: clean
clean:
	rm -rf bin $(DIST)

.PHONY: dist
dist:
	@mkdir -p $(DIST)
	GOOS=linux GOARCH=amd64 go build -o $(DIST)/unifi-dns-sync-linux-amd64 $(CMD)
	GOOS=linux GOARCH=arm64 go build -o $(DIST)/unifi-dns-sync-linux-arm64 $(CMD)
	GOOS=darwin GOARCH=arm64 go build -o $(DIST)/unifi-dns-sync-darwin-arm64 $(CMD)
	GOOS=darwin GOARCH=amd64 go build -o $(DIST)/unifi-dns-sync-darwin-amd64 $(CMD)