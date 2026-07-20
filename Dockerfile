FROM golang:1.24-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN go build -o /banner-rotation ./cmd/banner-rotation

FROM alpine:3.19

WORKDIR /app

COPY --from=builder /banner-rotation .
COPY configs/ configs/

ENV CONFIG_PATH=configs/config.yaml

EXPOSE 8080

ENTRYPOINT ["/app/banner-rotation"]
