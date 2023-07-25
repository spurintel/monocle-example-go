GOOS=$(shell go env GOOS)
GOARCH=$(shell go env GOARCH)

include .env
export

all: format bin

bin:
	go mod download
	go build -a -o server_${GOARCH}_${GOOS} main.go

bin-linux: GOOS=linux
bin-linux: GOARCH=amd64
bin-linux:
	go mod download
	env GOOS=${GOOS} GOARCH=${GOARCH} go build -o server_${GOARCH}_${GOOS} main.go

format:
	go fmt ./...

run: bin
run:
	./server_${GOARCH}_${GOOS} --port=9090