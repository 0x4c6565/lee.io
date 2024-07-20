FROM golang:1.22.5-alpine3.20 AS builder
WORKDIR /build
COPY . .
RUN go build -o lee.io

FROM alpine:3.20
WORKDIR /app
COPY --from=builder /build/lee.io .
COPY static static
ENTRYPOINT [ "/app/lee.io" ]