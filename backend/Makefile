.PHONY: build run tidy vet

build:
	go build -o kikundi-api .

run:
	go run .

tidy:
	GONOSUMCHECK=* GONOSUMDB=* go mod tidy

vet:
	go vet ./...

test:
	go test ./...
