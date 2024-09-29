FROM golang:1.21-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /ai-cli ./cmd/ai-cli

FROM alpine:3.18

WORKDIR /root/

COPY --from=builder /ai-cli .

CMD ["./ai-cli"]
