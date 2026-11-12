# Stage 1: Compile Go Quantitative Engine to WebAssembly (offline/local-first fallback)
FROM golang:1.24-alpine AS wasm-builder
WORKDIR /app
COPY go.mod ./
COPY cmd/ ./cmd/
COPY internal/ ./internal/
RUN mkdir -p /out && \
    GOOS=js GOARCH=wasm go build -trimpath -ldflags="-s -w" -o /out/firesage.wasm ./cmd/firesage-wasm && \
    (cp "$(go env GOROOT)/lib/wasm/wasm_exec.js" /out/wasm_exec.js || cp "$(go env GOROOT)/misc/wasm/wasm_exec.js" /out/wasm_exec.js)

# Stage 2: Build the React 19 + TypeScript + Vite Interactive Visualization Suite
FROM node:22-alpine AS web-builder
WORKDIR /web
COPY web/package*.json ./
RUN npm ci
COPY web/ ./
COPY --from=wasm-builder /out/firesage.wasm ./public/firesage.wasm
COPY --from=wasm-builder /out/wasm_exec.js ./public/wasm_exec.js
RUN npm run build

# Stage 3: Build the static Go concurrent Monte Carlo & HTTP binary with embedded SPA
FROM wasm-builder AS go-builder
COPY --from=web-builder /web/dist ./internal/server/dist
RUN CGO_ENABLED=0 GOOS=linux go build -trimpath -ldflags="-s -w" -o /firesage ./cmd/firesage

# Stage 4: Minimal distroless Google Cloud Run container
FROM gcr.io/distroless/static-debian12:nonroot
WORKDIR /
COPY --from=go-builder /firesage /firesage
ENV PORT=8080
ENV HOST=0.0.0.0
EXPOSE 8080
USER nonroot:nonroot
ENTRYPOINT ["/firesage"]
