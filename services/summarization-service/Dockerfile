FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o /bin/summarization-service ./cmd

FROM debian:bookworm-slim
RUN apt-get update && apt-get install -y --no-install-recommends \
    ca-certificates \
    curl \
    xz-utils \
    fontconfig \
    fonts-inter \
    fonts-linuxlibertine \
    && fc-cache -fv \
    && curl -fsSL https://github.com/typst/typst/releases/download/v0.14.2/typst-x86_64-unknown-linux-musl.tar.xz \
       | tar -xJ --strip-components=1 -C /usr/local/bin typst-x86_64-unknown-linux-musl/typst \
    && rm -rf /var/lib/apt/lists/*

COPY --from=builder /bin/summarization-service /usr/local/bin/summarization-service

ENTRYPOINT ["/usr/local/bin/summarization-service"]
