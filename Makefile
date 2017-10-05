VERSION := $(shell git describe --tags)

.PHONY: setup test build

setup:
	go get github.com/aws/aws-sdk-go
	go get github.com/comail/colog

test:
	go test

build:
	cd cmd/heuristic-autoscaling && go build -ldflags "-X main.version=${VERSION}" -o ../../bin/heuristic-autoscaling
