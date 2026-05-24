FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /kazi-server ./cmd/server

FROM alpine:latest

WORKDIR /app

RUN apk add --no-cache dos2unix

RUN mkdir -p /data

COPY --from=builder /kazi-server .
COPY migrations ./migrations
COPY entrypoint.sh /

RUN dos2unix /entrypoint.sh && chmod +x /entrypoint.sh

EXPOSE 8080

ENTRYPOINT ["/entrypoint.sh"]
CMD ["/app/kazi-server"]