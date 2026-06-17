FROM golang:1.25-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -o transcription-service ./cmd/main.go

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/transcription-service .
EXPOSE 8080
ENTRYPOINT ["./transcription-service"]