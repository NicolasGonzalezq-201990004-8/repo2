FROM golang:1.25-alpine AS builder

# Usa la carpeta del proyecto como raíz de módulo; Go trabaja en modo módulo
# por defecto y las imports locales "lab_3/..." se resuelven sin depender de
# GOPATH.
WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o /app/broker ./broker

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/broker /app/broker

COPY flight_updates.csv .

EXPOSE 50050 

CMD ["/app/broker"]