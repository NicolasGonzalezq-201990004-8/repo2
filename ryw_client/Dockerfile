FROM golang:1.25-alpine AS builder
WORKDIR /src
ENV GO111MODULE=on
COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto

COPY ryw_client ./ryw_client

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/ryw_client ./ryw_client

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/ryw_client /app/ryw_client


COPY flight_updates.csv . 

CMD ["/app/ryw_client"]