package main

import (
	"log"
	"net"
	"os"

	dpb "lab_3/proto/datanode/dpb"
	"google.golang.org/grpc"
)

func main() {
	port := os.Getenv("DATANODE_PORT")
	if port == "" {
		log.Fatal("Debe definir DATANODE_PORT")
	}
	if port[0] != ':' {
		port = ":" + port
	}

	lis, err := net.Listen("tcp", port)
	if err != nil {
		log.Fatalf("Error al escuchar en %s: %v", port, err)
	}

	server := grpc.NewServer()
	dn := NewDatanodeServer()

	dpb.RegisterDatanodeServiceServer(server, dn)
	log.Printf("[Datanode %s] iniciado en %s", dn.id, port)

	log.Printf("[Datanode %s] Gossip periódico activado", dn.id)
	go dn.StartPeriodicGossip()

	if err := server.Serve(lis); err != nil {
		log.Fatalf("Error en gRPC: %v", err)
	}
}
