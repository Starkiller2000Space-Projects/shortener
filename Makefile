build:
	go build -o bin/shortener ./cmd/shortener

test:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html

run-client:
	go run ./cmd/client/main.go

run-binary:
	./bin/shortener

mocks:
	go generate ./...