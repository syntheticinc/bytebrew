FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN go build -o mock-mcp-http ./testdata/mock-mcp-http

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mock-mcp-http .
EXPOSE 9556
CMD ["./mock-mcp-http"]
