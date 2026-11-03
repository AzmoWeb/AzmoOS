VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO      := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: build build-all build-arm64 build-armv6 build-amd64 run test image clean

build:                      ## binaire pour la machine actuelle
	$(GO) -o dist/azmod ./cmd/azmod

build-arm64:                ## Pi 3/4/5/Zero 2 W (OS 64 bits)
	GOOS=linux GOARCH=arm64 $(GO) -o dist/azmod-linux-arm64 ./cmd/azmod

build-armv6:                ## Pi 1/2/Zero/Zero W et OS 32 bits
	GOOS=linux GOARCH=arm GOARM=6 $(GO) -o dist/azmod-linux-armv6 ./cmd/azmod

build-amd64:                ## PC / mini-PC x86_64
	GOOS=linux GOARCH=amd64 $(GO) -o dist/azmod-linux-amd64 ./cmd/azmod

build-all: build-arm64 build-armv6 build-amd64

run: build                  ## lancement local de développement sur :8080
	mkdir -p .dev/data .dev/conf
	./dist/azmod -addr :8080 -data .dev/data -config .dev/conf

test:
	go vet ./... && go test ./...

image:                      ## image Raspberry Pi .img.xz (pi-gen)
	./image/build-image.sh

clean:
	rm -rf dist .dev
