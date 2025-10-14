FROM golang:1.25.2-trixie AS builder
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o delayed-notifier ./cmd

FROM alpine:3.20
WORKDIR /app

RUN apk add --no-cache ca-certificates netcat-openbsd

COPY docker/app/wait-for-deps.sh /usr/local/bin/wait-for-deps

RUN chmod +x /usr/local/bin/wait-for-deps

COPY --from=builder /app/delayed-notifier /app/delayed-notifier

ENV HTTP_PORT=8080
EXPOSE 8080

ENTRYPOINT ["/usr/local/bin/wait-for-deps"]
CMD ["/app/delayed-notifier"]
