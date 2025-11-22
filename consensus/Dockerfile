FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GO111MODULE=on
ENV GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/consensusnode ./consensus

FROM alpine:latest

WORKDIR /app

# Instalar en el $PATH para evitar conflictos con directorios llamados
# "/app/consensusnode".
COPY --from=builder /app/bin/consensusnode /usr/local/bin/consensusnode

EXPOSE 50060

ENTRYPOINT ["consensusnode"]
