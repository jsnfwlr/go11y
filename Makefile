.PHONY: test tools
tools:
	@go install github.com/mfridman/tparse@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

test:
	@echo "Running tests..."
	go test ./... -json -cover -count 1 | tparse -notests  --progress --pass
