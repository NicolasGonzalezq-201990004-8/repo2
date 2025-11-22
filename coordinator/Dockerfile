FROM golang:1.25-alpine AS builder

WORKDIR /src

COPY go.mod go.sum ./
RUN go mod download

COPY proto ./proto

COPY coordinator ./coordinator

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/coordinator ./coordinator

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/coordinator /app/coordinator

EXPOSE 50070

CMD ["/app/coordinator"]