FROM golang:1.25-alpine AS builder

# Keep the source under GOPATH so module imports resolve even if module mode is off.
WORKDIR /go/src/lab_3

ENV GO111MODULE=on \
    GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN CGO_ENABLED=0 GOOS=linux go build -o /app/consensusnode ./consensus

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/consensusnode /app/consensusnode

EXPOSE 50060

CMD ["/app/consensusnode"]