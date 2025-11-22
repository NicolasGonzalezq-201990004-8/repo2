package main

import (
	"context"
	"log"
	"net"
	"os"

	bpb "lab_3/proto/broker/bpb"

	"google.golang.org/grpc"
)

func main() {
	brokerPort := os.Getenv("BROKER_PORT")
	if brokerPort == "" {
		log.Fatal("La variable de entorno BROKER_PORT no está configurada.")
	}
	if brokerPort[0] != ':' {
		brokerPort = ":" + brokerPort
	}
	log.Printf("Iniciando Broker en puerto %s", brokerPort)

	lis, err := net.Listen("tcp", brokerPort)
	if err != nil {
		log.Fatalf("Falló al escuchar en %s: %v", brokerPort, err)
	}

	s := grpc.NewServer()

	// FASE 1
	brokerServer := NewBrokerServer()

	bpb.RegisterBrokerServiceServer(s, brokerServer)

	// FASE 2
	log.Println("[Broker] Iniciando simulación de eventos...")
	go brokerServer.StartEventSimulation(context.Background())

	go brokerServer.ReportWriter()

	log.Printf("[Broker] Servidor gRPC listo y escuchando en puerto %s.", brokerPort)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error en el servidor gRPC: %v", err)
	}
}
