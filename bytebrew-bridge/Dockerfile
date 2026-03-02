FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o /bridge ./cmd/bridge

FROM alpine:3.21
COPY --from=builder /bridge /bridge
EXPOSE 8443
ENTRYPOINT ["/bridge"]
