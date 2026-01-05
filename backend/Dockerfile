FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
RUN go mod init monitor && go build -o stream-monitor main.go

FROM alpine:latest
RUN apk --no-cache add wget ca-certificates
WORKDIR /root/
COPY --from=builder /app/stream-monitor .
EXPOSE 8080
CMD ["./stream-monitor"]
