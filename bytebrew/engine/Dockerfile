FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bytebrew-ce ./cmd/ce

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bytebrew-ce /usr/local/bin/
ENTRYPOINT ["bytebrew-ce"]
CMD ["--config", "/etc/bytebrew/bytebrew.yaml"]
