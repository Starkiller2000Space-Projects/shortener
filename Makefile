build:
	go build -o bin/shortener ./cmd/shortener

test:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | grep total

run-client:
	go run ./cmd/client/main.go

run-binary:
	./bin/shortener

mocks:
	go generate ./...