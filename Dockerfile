FROM golang:1.26-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .
RUN CGO_ENABLED=0 GOOS=linux go build -ldflags="-s -w" -o /vagoubot ./cmd/scraper

FROM alpine:3.20

RUN apk add --no-cache ca-certificates tzdata

WORKDIR /app

COPY --from=builder /vagoubot /app/vagoubot
COPY config.yaml /app/config.yaml

RUN mkdir -p /app/data
ENV DATA_DIR=/app/data
ENV TZ=America/Sao_Paulo

CMD ["/app/vagoubot"]