VERSION    := 0.1
COMMIT_SHA := $(shell git rev-parse --short HEAD)

all: build

build:
	# go get -uv
	go build -buildmode=c-shared -o pam_beacon.so

install: build
	cp pam_beacon.so /usr/lib/security/

test:
	go test ./... -v

fmt:
	go fmt ./... -v

.PHONY: install test fmt
