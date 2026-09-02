.PHONY: build run test vet fmt tidy docker clean

BINARY := bin/taskflow

build:
	go build -o $(BINARY) ./cmd/taskflow

run:
	go run ./cmd/taskflow

test:
	go test ./... -race -cover

vet:
	go vet ./...

fmt:
	gofmt -w .

tidy:
	go mod tidy

docker:
	docker build -t taskflow:latest .

clean:
	rm -rf bin
