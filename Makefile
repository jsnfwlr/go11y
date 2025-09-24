.PHONY: test tools simple-demo server-demo client-demo
tools:
	@go install github.com/mfridman/tparse@latest
	@go install github.com/sqlc-dev/sqlc/cmd/sqlc@latest

test:
	@echo "Running tests..."
	@go test ./... -json -cover -count 1 | tparse -progress -trimpath "github.com/jsnfwlr/go11y" -all

coverage:
	@echo "Generating coverage report..."
	@go test ./... -coverprofile=coverage.out
	@go tool cover -html=coverage.out -o coverage.html
	@rm coverage.out

simple-demo:
	cd demo; ENV=demo go run . -simple


server-demo:
	cd demo; ENV=demo go run . -server

client-demo:
	cd demo; ENV=demo go run .
