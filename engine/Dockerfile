FROM golang:1.24-alpine AS builder
WORKDIR /app
COPY go.mod go.sum ./
RUN go mod download
COPY . .
RUN CGO_ENABLED=0 go build -o bytebrew-ce ./cmd/ce

FROM node:22-alpine AS admin-builder
WORKDIR /admin
COPY admin/package.json admin/package-lock.json ./
RUN npm ci
COPY admin/ .
RUN npm run build

FROM alpine:3.19
RUN apk --no-cache add ca-certificates
COPY --from=builder /app/bytebrew-ce /usr/local/bin/
COPY --from=admin-builder /admin/dist /usr/share/bytebrew/admin
EXPOSE 8443
ENTRYPOINT ["bytebrew-ce"]
