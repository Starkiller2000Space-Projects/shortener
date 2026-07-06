CWD := $(CURDIR)

test:
	go test -coverprofile=coverage.out ./...
	go tool cover -func=coverage.out | grep total
	go tool cover -html=coverage.out -o coverage.html

run-client:
	go run ./cmd/client/main.go

run:
	go build -o bin/shortener ./cmd/shortener
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
