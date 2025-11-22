FROM golang:1.25-alpine AS builder

# Usa la carpeta del proyecto como raíz de módulo; Go trabaja en modo módulo
# por defecto y las imports locales "lab_3/..." se resuelven sin depender de
# GOPATH.
WORKDIR /app

ENV GO111MODULE=on
ENV GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/broker ./broker

FROM alpine:latest

WORKDIR /app

# Instala el binario en una ruta del $PATH para evitar cualquier choque con
# directorios llamados "/app/broker".
COPY --from=builder /app/bin/broker /usr/local/bin/broker

COPY flight_updates.csv .

EXPOSE 50050

ENTRYPOINT ["broker"]
