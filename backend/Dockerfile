FROM golang:1.21-alpine AS builder
WORKDIR /app
COPY main.go .
# go mod init'i otomatik yapıyoruz:
RUN go mod init monitor && go build -o stream-monitor main.go

FROM alpine:latest
WORKDIR /root/
COPY --from=builder /app/stream-monitor .
CMD ["./stream-monitor"]