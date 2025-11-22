FROM golang:1.25-alpine AS builder
WORKDIR /app

ENV GO111MODULE=on
ENV GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/ryw_client ./ryw_client

FROM alpine:latest
WORKDIR /app

# Instalar en el $PATH para evitar conflictos con directorios llamados
# "/app/ryw_client".
COPY --from=builder /app/bin/ryw_client /usr/local/bin/ryw_client


COPY flight_updates.csv .

ENTRYPOINT ["ryw_client"]
