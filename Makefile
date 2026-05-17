build:
	go build -o bin/shortener ./cmd/shortener

test:
	go test -cover ./...

run-client:
	go run ./cmd/client/main.go

run-binary:
	./bin/shortener

mocks:
	go generate ./...