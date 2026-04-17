# Build stage
FROM golang:1.22-alpine AS builder

WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o app ./cmd/api

# Run stage
FROM alpine:latest

WORKDIR /app
COPY --from=builder /app/app .

ENV PORT=8080

EXPOSE 8080

CMD ["./app"]