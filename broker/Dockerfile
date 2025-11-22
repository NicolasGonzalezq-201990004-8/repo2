FROM golang:1.25-alpine AS builder

# Place the source tree at a GOPATH-compatible location so imports such as
# "lab_3/proto/..." resolve even if module mode is accidentally disabled.
WORKDIR /go/src/lab_3

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