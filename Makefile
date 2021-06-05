VERSION    := 0.2.0
COMMIT_SHA := $(shell git rev-parse --short HEAD)

all: build

build:
	go build -buildmode=c-shared -o pam_beacon.so

install:
	cp pam_beacon.so /usr/lib/security/

.PHONY: deps build install test fmt
