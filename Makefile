.PHONY: wasm web build test clean

GO ?= go
NPM ?= npm

wasm:
	GOOS=js GOARCH=wasm $(GO) build -trimpath -ldflags="-s -w" -o web/public/firesage.wasm ./cmd/firesage-wasm
	cp "$$($(GO) env GOROOT)/lib/wasm/wasm_exec.js" web/public/wasm_exec.js

web: wasm
	$(NPM) --prefix web run build

build: web
	cp -R web/dist/* internal/server/dist/
	CGO_ENABLED=0 $(GO) build -trimpath -ldflags="-s -w" -o firesage ./cmd/firesage

test:
	$(GO) test -v ./internal/...

clean:
	rm -rf web/dist firesage web/public/firesage.wasm web/public/wasm_exec.js
	find internal/server/dist -mindepth 1 ! -name 'index.html' -delete
