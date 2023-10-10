FROM golang:1.21 AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build cmd/proxy.go

FROM alpine
WORKDIR /app

COPY --from=builder /app/proxy .

CMD ["./proxy"]


