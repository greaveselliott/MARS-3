.PHONY: build test vet verify

build:
	go build ./cmd/mars3

test:
	go test ./...

vet:
	go vet ./...

verify: test vet
	go run ./cmd/mars3 doctrine check --repo .
	go run ./cmd/mars3 plan check --repo .
	go run ./cmd/mars3 docsync audit --repo .
	go run ./cmd/mars3 public-check --repo .
	git diff --check
	gitleaks detect --no-git --source . --redact --no-banner
	gitleaks detect --source . --redact --no-banner
