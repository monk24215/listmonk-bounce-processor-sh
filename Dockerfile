FROM golang:1.21-alpine AS builder

WORKDIR /build

COPY go.mod .
COPY main.go .

RUN go mod tidy && CGO_ENABLED=0 GOOS=linux GOARCH=amd64 go build -o bounce-processor .

FROM alpine:latest

RUN apk --no-cache add ca-certificates

WORKDIR /app

COPY --from=builder /build/bounce-processor .

RUN chmod +x ./bounce-processor

CMD ["./bounce-processor"]
