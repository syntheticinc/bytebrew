FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o mcp-docs-server .

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/mcp-docs-server /usr/local/bin/
EXPOSE 8090
ENTRYPOINT ["mcp-docs-server"]
