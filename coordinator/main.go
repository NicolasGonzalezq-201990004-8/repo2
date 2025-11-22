package main

import (
	"log"
	"net"
	"os"
	"strings"

	cdpb "lab_3/proto/coordinator/cdpb"

	"google.golang.org/grpc"
)

func main() {
	port := os.Getenv("COORDINATOR_PORT")
	if port == "" {
		log.Fatal("COORDINATOR_PORT no configurado")
	}

	listenAddr := port
	if !strings.Contains(listenAddr, ":") {
		listenAddr = ":" + listenAddr
	}

	log.Printf("Iniciando Coordinator en %s...", listenAddr)

	lis, err := net.Listen("tcp", listenAddr)
	if err != nil {
		log.Fatalf("Error al escuchar: %v", err)
	}

	coordServer := NewCoordinatorServer()
	s := grpc.NewServer()

	cdpb.RegisterCoordinatorServiceServer(s, coordServer)

	log.Printf("[Coordinator] Servidor listo.")
	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error en servidor gRPC: %v", err)
	}

}
