mod: go mod download

test:
	go test -race ./...

lint:
	go vet ./...

build:
	go build -o ./bin/muzz