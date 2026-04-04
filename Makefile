build:
	go build -o bin/shortener ./cmd/shortener

test:
	go test ./...

run_client:
	go run ./cmd/client/main.go

run_binary:
	./bin/shortener
