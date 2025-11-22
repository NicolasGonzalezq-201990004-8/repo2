# ---------- STAGE 1: BUILD ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

ENV GO111MODULE=on
ENV GOFLAGS=-mod=mod

# Copiar los archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar TODO el proyecto
COPY . .

# Compilar SOLO el datanode
RUN mkdir -p /app/bin \
    && CGO_ENABLED=0 GOOS=linux go build -o /app/bin/datanode ./datanode


# ---------- STAGE 2: RUNTIME ----------
FROM alpine:latest

WORKDIR /app

# Instalar el binario en el $PATH para evitar conflictos con directorios
# llamados "/app/datanode".
COPY --from=builder /app/bin/datanode /usr/local/bin/datanode

# Exponer puerto del datanode (puede variar)
EXPOSE 50051

ENTRYPOINT ["datanode"]

