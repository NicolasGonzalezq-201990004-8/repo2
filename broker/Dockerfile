FROM golang:1.25-alpine AS builder

WORKDIR /src

ENV GO111MODULE=on \
    GOFLAGS=-mod=mod

COPY go.mod go.sum ./

RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/broker ./broker

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/broker /app/broker

COPY flight_updates.csv .

EXPOSE 50050 

CMD ["/app/broker"]