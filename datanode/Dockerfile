# ---------- STAGE 1: BUILD ----------
FROM golang:1.25-alpine AS builder

WORKDIR /app

# Copiar los archivos de dependencias
COPY go.mod go.sum ./
RUN go mod download

# Copiar TODO el proyecto
COPY . .

# Compilar SOLO el datanode
RUN GO111MODULE=on CGO_ENABLED=0 GOOS=linux go build -o /app/datanode ./datanode


# ---------- STAGE 2: RUNTIME ----------
FROM alpine:latest

WORKDIR /app

# Copiar binario
COPY --from=builder /app/datanode /app/datanode

# Exponer puerto del datanode (puede variar)
EXPOSE 50051

CMD ["/app/datanode"]
