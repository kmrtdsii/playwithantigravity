# Build Stage
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Install build tools if needed
# (alpine default git might be enough, but lets be sure)
RUN apk add --no-cache git

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o server ./cmd/server

# Run Stage
FROM alpine:latest

WORKDIR /app

# Install runtime dependencies (if any)
RUN apk add --no-cache ca-certificates

COPY --from=builder /app/server .

EXPOSE 8080

CMD ["./server"]
