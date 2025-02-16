# syntax=docker/dockerfile:1
FROM golang:1.24 AS builder
WORKDIR /go/github.com/whywaita/rfid-poker

COPY ["./", "./"]

ENV GOOS=linux
ENV GOARCH=amd64
ENV CGO_ENABLED=0

RUN make build

FROM alpine:latest
COPY --from=builder /go/github.com/whywaita/rfid-poker/bin/cmd /usr/local/bin/rfid-poker

CMD ["/usr/local/bin/rfid-poker"]