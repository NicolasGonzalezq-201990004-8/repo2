FROM golang:1.25-alpine AS builder

WORKDIR /app

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o /app/consensusnode ./consensus

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/consensusnode /app/consensusnode

EXPOSE 50060

CMD ["/app/consensusnode"]