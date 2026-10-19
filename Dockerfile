# syntax=docker/dockerfile:1

# Stage 1: build the core (CGO-free) binary.
FROM golang:1.26-alpine AS builder

WORKDIR /src

# Cache module downloads.
COPY go.mod go.sum ./
RUN go mod download

# Copy the source.
COPY . .

# Build a static, stripped binary for the target platform.
ARG TARGETOS=linux
ARG TARGETARCH=amd64
ENV CGO_ENABLED=0 GOOS=$TARGETOS GOARCH=$TARGETARCH
RUN go build -trimpath -ldflags "-s -w" -o /out/gospan ./cmd/gospan

# Stage 2: minimal runtime image.
FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /out/gospan /usr/local/bin/gospan
COPY configs/ /app/configs/

# Default config location inside the container.
ENV GOSPAN_CONFIG=/app/configs/config.example.yaml

ENTRYPOINT ["/usr/local/bin/gospan"]
CMD ["version"]
