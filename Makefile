build:
	go build -o bin/shortener ./cmd/shortener

test:
	go test ./...
