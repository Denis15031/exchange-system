FROM golang:1.25-bookworm AS builder
WORKDIR /build

COPY go.work ./
COPY */go.mod */go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build \
    -ldflags="-w -s" \
    -o /app/server ${MAIN_PATH}

FROM debian:bookworm-slim
RUN apt-get update && \
    apt-get install -y --no-install-recommends ca-certificates wget && \
    apt-get clean && rm -rf /var/lib/apt/lists/* && \
    groupadd -g 1001 app && \
    useradd -u 1001 -g app -m -s /bin/bash app

WORKDIR /app
COPY --from=builder /app/server .
USER app

HEALTHCHECK --interval=15s --timeout=3s --start-period=10s --retries=3 \
  CMD wget -qO- http://localhost:9094/healthz || exit 1

ENTRYPOINT ["./server"]