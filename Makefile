# Azmo OS — Made by Azmo
VERSION ?= $(shell git describe --tags --always --dirty 2>/dev/null || echo dev)
LDFLAGS := -s -w -X main.version=$(VERSION)
GO      := CGO_ENABLED=0 go build -trimpath -ldflags "$(LDFLAGS)"

.PHONY: help web build build-all build-arm64 build-armv6 build-amd64 dev run test lint release image clean

help:                       ## affiche cette aide
	@grep -E '^[a-z-]+:.*##' $(MAKEFILE_LIST) | awk -F':.*## ' '{printf "  \033[35m%-12s\033[0m %s\n", $$1, $$2}'

frontend/node_modules: frontend/package-lock.json
	cd frontend && npm ci
	@touch $@

web: frontend/node_modules  ## compile l'interface React → web/dist
	cd frontend && npm run build

build: web                  ## binaire pour cette machine (dist/azmod)
	$(GO) -o dist/azmod ./cmd/azmod

build-arm64:                ## Pi 3/4/5/Zero 2 W — OS 64 bits
	GOOS=linux GOARCH=arm64 $(GO) -o dist/azmod-linux-arm64 ./cmd/azmod

build-armv6:                ## Pi 1/2/Zero — OS 32 bits
	GOOS=linux GOARCH=arm GOARM=6 $(GO) -o dist/azmod-linux-armv6 ./cmd/azmod

build-amd64:                ## PC / mini-PC x86_64
	GOOS=linux GOARCH=amd64 $(GO) -o dist/azmod-linux-amd64 ./cmd/azmod

build-all: web build-arm64 build-armv6 build-amd64

run: build                  ## lance Azmo OS en local sur :8080 (données dans .dev/)
	mkdir -p .dev/data .dev/conf
	./dist/azmod -addr :8080 -data .dev/data -config .dev/conf -web $(CURDIR)/web/dist

dev:                        ## interface en rechargement à chaud (lancer « make run » à côté)
	cd frontend && npm run dev

test:                       ## tests Go + vérification TypeScript + traductions
	go vet ./...
	go test ./...
	cd frontend && npx tsc -b
	cd frontend && npm run i18n:check

release: build-all          ## archives de publication dans dist/
	cd web/dist && tar --exclude=.keep -czf ../../dist/azmoos-web.tar.gz .
	cd dist && sha256sum azmod-linux-* azmoos-web.tar.gz > SHA256SUMS

image:                      ## image Raspberry Pi .img.xz (pi-gen, ~45 min)
	./image/build-image.sh

clean:
	rm -rf dist .dev frontend/node_modules
	find web/dist -mindepth 1 ! -name .keep -delete
