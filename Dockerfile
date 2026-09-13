FROM golang:1.27 AS builder

WORKDIR /workspace

COPY go.mod go.sum ./
RUN go mod download

COPY . .

ENV CGO_ENABLED=0

RUN go build -o /workspace/ecommerce ./cmd/api

FROM alpine:3.22

WORKDIR /workspace

COPY --from=builder /workspace/ecommerce .

EXPOSE 8080

CMD ["./ecommerce"]