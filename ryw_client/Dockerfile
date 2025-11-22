FROM golang:1.25-alpine AS builder
WORKDIR /go/src/lab_3
# Fuerza el modo módulo y deshabilita workspaces para evitar que Go vuelva al
# GOPATH al resolver imports locales durante la construcción.
ENV GO111MODULE=on \
    GOWORK=off \
    GOFLAGS=-mod=mod
COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0 GOOS=linux go build -o /app/ryw_client ./ryw_client

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ryw_client /app/ryw_client


COPY flight_updates.csv . 

CMD ["/app/ryw_client"]