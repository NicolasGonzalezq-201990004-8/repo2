package main

import (
	"log"
	"math/rand"
	"net"
	"os"
	"time"

	cspb "lab_3/proto/consensus/cspb"

	"google.golang.org/grpc"
)

func main() {

	rand.Seed(time.Now().UnixNano())

	port := os.Getenv("CONSENSUS_PORT")
	if port == "" {
		log.Fatal("La variable de entorno CONSENSUS_PORT no está configurada.")
	}

	if port[0] != ':' {
		port = ":" + port
	}

	log.Printf("Iniciando Nodo de Consenso en puerto %s...", port)

	// Inicializar el Listener TCP
	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Falló al escuchar en %s: %v", port, err)
	}

	raftNode := NewConsensusNode()

	// Conectar a los Pares
	raftNode.ConnectToPeers()

	// Timer de Elección
	go raftNode.StartElectionTimer()

	s := grpc.NewServer()

	consensusServer := NewConsensusServer(raftNode)

	cspb.RegisterConsensusServiceServer(s, consensusServer)
	cspb.RegisterInternalConsensusServiceServer(s, consensusServer)

	log.Printf("[%s] Servidor gRPC listo y escuchando (Estado: %s)", raftNode.ID, raftNode.State)

	if err := s.Serve(lis); err != nil {
		log.Fatalf("Error crítico en servidor gRPC: %v", err)
	}
}