FROM golang:1.26-alpine3.22 AS builder
WORKDIR /build
COPY . .
RUN go build -o lee.io

FROM alpine:3.22
WORKDIR /app
COPY --from=builder /build/lee.io .
COPY static static
ENTRYPOINT [ "/app/lee.io" ]