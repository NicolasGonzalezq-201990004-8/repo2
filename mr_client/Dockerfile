FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GO111MODULE=on
ENV GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/mr_client ./mr_client

FROM alpine:latest
WORKDIR /app

# Instalar en el $PATH para evitar conflictos con directorios llamados
# "/app/mr_client".
COPY --from=builder /app/bin/mr_client /usr/local/bin/mr_client

COPY flight_updates.csv .

ENTRYPOINT ["mr_client"]

