CWD := $(CURDIR)

lint:
	gofmt -w .
	goimports -w .

static-lint:
	go run ./cmd/staticlint/main.go ./cmd/...
	go run ./cmd/staticlint/main.go ./internal/...

test:
	go test -coverprofile=coverage.out ./internal/...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html

doc:
	go doc -http :6060

run-client:
	go run ./cmd/client/main.go


VERSION := $(shell git describe --tags --always 2>/dev/null || echo "")
DATE    := $(shell date -u +'%Y-%m-%dT%H:%M:%SZ')
COMMIT  := $(shell git rev-parse HEAD 2>/dev/null || echo "")

run:
	go build -ldflags "-X main.buildVersion=$(VERSION) \
                   -X main.buildDate=$(DATE) \
                   -X main.buildCommit=$(COMMIT)" \
                   -o bin/shortener ./cmd/shortener
	./bin/shortener

mocks:
	go generate ./...

bench:
	go test -bench=. -benchmem ./...

load:
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s -s shorten.lua http://host.docker.internal:8080/ &
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s -s shorten-json.lua http://host.docker.internal:8080/api/shorten &
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s -s batch.lua http://host.docker.internal:8080/api/shorten/batch &
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s http://host.docker.internal:8080/ping &
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s http://host.docker.internal:8080/tbC-l-Xy &  # requires existing id
	docker run --rm -v $(CWD)/scripts/wrk/:/data skandyla/wrk -t8 -c100 -d30s http://host.docker.internal:8080/api/user/urls &
	wait

pprof-%:
	mkdir -p profiles
	curl -s -o profiles/$*.pprof "http://localhost:6060/debug/pprof/heap"

pprof-watch-%:
	go tool pprof profiles/$*.pprof

pprof-compare:
	go tool pprof -top -diff_base=profiles/base.pprof -alloc_space profiles/result.pprof

resets:
	go run ./cmd/reset/main.go

proto:
	protoc --go_out=. --go_opt=paths=source_relative --go-grpc_out=. --go-grpc_opt=paths=source_relative api/shortener.proto
