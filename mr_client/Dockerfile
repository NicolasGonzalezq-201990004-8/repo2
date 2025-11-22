FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o /app/mr_client ./mr_client

FROM alpine:latest
WORKDIR /app
COPY --from=builder /app/mr_client /app/mr_client

COPY flight_updates.csv . 

CMD ["/app/mr_client"]
