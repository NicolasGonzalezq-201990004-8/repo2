FROM golang:1.25-alpine AS builder

WORKDIR /go/src/lab_3

# Fuerza el modo módulo y desactiva workspaces para evitar que Go resuelva
# imports en GOPATH durante la compilación.
ENV GO111MODULE=on \
    GOWORK=off \
    GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0 GOOS=linux go build -o /app/coordinator ./coordinator

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/coordinator /app/coordinator

EXPOSE 50070

CMD ["/app/coordinator"]