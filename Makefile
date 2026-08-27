.PHONY: build test test-authority-postgres vet verify

build:
	go build ./cmd/mars3

test:
	go test ./...

test-authority-postgres:
	go test ./internal/authority/postgres -run '^TestPostgresLeaseLifecycleAndRestart$$' -count=1 -v

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
