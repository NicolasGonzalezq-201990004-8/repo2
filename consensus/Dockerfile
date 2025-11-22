FROM golang:1.25-alpine AS builder

# Keep the source under GOPATH so module imports resolve even if module mode is off.
WORKDIR /go/src/lab_3

# Fuerza el modo módulo y deshabilita workspaces que puedan interferir con las
# imports locales durante la compilación.
ENV GO111MODULE=on \
    GOWORK=off \
    GOFLAGS=-mod=mod

COPY go.mod go.sum ./
RUN go mod download

COPY . .

RUN GO111MODULE=on GOWORK=off GOFLAGS=-mod=mod CGO_ENABLED=0 GOOS=linux go build -o /app/consensusnode ./consensus

FROM alpine:latest

WORKDIR /app

COPY --from=builder /app/consensusnode /app/consensusnode

EXPOSE 50060

CMD ["/app/consensusnode"]