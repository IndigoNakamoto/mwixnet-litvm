# Build from repo root: docker build -f mln-sidecar/Dockerfile -t mln-sidecar:local mln-sidecar

FROM golang:1.22-bookworm AS builder
WORKDIR /src
COPY go.mod go.sum* ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -trimpath -ldflags="-s -w" -o /out/mln-sidecar ./cmd/mln-sidecar

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends ca-certificates \
  && rm -rf /var/lib/apt/lists/*
COPY --from=builder /out/mln-sidecar /usr/local/bin/mln-sidecar
EXPOSE 8080
ENTRYPOINT ["/usr/local/bin/mln-sidecar"]
CMD ["-port", "8080", "-mode", "mock"]
