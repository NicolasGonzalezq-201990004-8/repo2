FROM golang:1.25-alpine AS builder

WORKDIR /go/src/lab_3

# Fuerza modo módulo y deshabilita workspaces para garantizar que las imports
# locales se resuelvan durante la construcción en contenedores.
ENV GO111MODULE=on \
    GOWORK=off \
    GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0 GOOS=linux go build -o /app/mr_client ./mr_client

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mr_client /app/mr_client

COPY flight_updates.csv . 

CMD ["/app/mr_client"]
