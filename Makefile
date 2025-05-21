VERSION    := 0.2.0
COMMIT_SHA := $(shell git rev-parse --short HEAD)
# For Ubuntu family linux distributions, try specifying /usr/lib/x86_64-linux-gnu
INSTALL_BASE := /usr/lib

all: build

build:
	go build -buildmode=c-shared -o pam_beacon.so

install:
	cp pam_beacon.so $(INSTALL_BASE)/security/

.PHONY: deps build install test fmt
