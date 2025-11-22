package main

import (
	"context"
	"log"
	"os"

	bpb "lab_3/proto/broker/bpb"
	cdpb "lab_3/proto/coordinator/cdpb"

	cmpb "lab_3/proto/common/cmpb"
	dpb "lab_3/proto/datanode/dpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type CoordinatorServer struct {
	cdpb.UnimplementedCoordinatorServiceServer
	sessions *SessionMap

	brokerClient bpb.BrokerServiceClient
}

func NewCoordinatorServer() *CoordinatorServer {
	brokerAddr := os.Getenv("BROKER_ADDR")
	if brokerAddr == "" {
		log.Fatal("BROKER_ADDR no configurada")
	}

	conn, err := grpc.Dial(brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		log.Fatalf("No se pudo conectar al Broker en %s: %v", brokerAddr, err)
	}

	log.Printf("[Coordinator] Conectado al Broker en %s", brokerAddr)

	return &CoordinatorServer{
		sessions:     NewSessionMap(),
		brokerClient: bpb.NewBrokerServiceClient(conn),
	}
}

func (s *CoordinatorServer) getDatanodeClient(addr string) (dpb.DatanodeServiceClient, *grpc.ClientConn, error) {
	conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
	if err != nil {
		return nil, nil, err
	}
	return dpb.NewDatanodeServiceClient(conn), conn, nil
}

func (s *CoordinatorServer) CheckIn(ctx context.Context, req *cmpb.CheckInRequest) (*cmpb.CheckInResponse, error) {
	log.Printf("[Coordinator] Iniciando transacción RYW para Vuelo %s", req.FlightId)
	dnResp, err := s.brokerClient.GetDatanode(ctx, &cmpb.Empty{})
	if err != nil {
		log.Printf("Error obteniendo Datanode del Broker: %v", err)
		return nil, err
	}
	targetAddr := dnResp.DatanodeId

	clientID := req.PassengerId
	s.sessions.RegisterAffinity(clientID, targetAddr)

	log.Printf("[Coordinator] Afinidad fijada: Cliente -> %s", targetAddr)

	req.TargetDatanodeId = targetAddr

	resp, err := s.brokerClient.CheckIn(ctx, req)
	if err != nil {
		log.Printf("Error en escritura vía Broker: %v", err)
		return nil, err
	}

	return resp, nil
}

func (s *CoordinatorServer) GetBoardingPass(ctx context.Context, req *cmpb.BoardingPassRequest) (*cmpb.BoardingPassResponse, error) {
	datanodeAddr, exists := s.sessions.CheckAffinity(req.PassengerId)

	if exists {
		log.Printf("[Coordinador] Sesión RYW encontrada para %s. Redirigiendo DIRECTO a %s", req.FlightId, datanodeAddr)

		// CONEXIÓN DIRECTA AL DATANODE (Bypass Broker)
		dnClient, conn, err := s.getDatanodeClient(datanodeAddr)
		if err != nil {
			log.Printf("Error conectando directo al Datanode %s: %v", datanodeAddr, err)
			return s.brokerClient.GetBoardingPass(ctx, req)
		}
		defer conn.Close()

		return dnClient.GetBoardingPass(ctx, req)

	} else {
		log.Printf("[Coordinador] Sin sesión para %s. Redirigiendo al Broker.", req.FlightId)
		//Broker
		return s.brokerClient.GetBoardingPass(ctx, req)
	}
}

func (s *CoordinatorServer) GetSeats(ctx context.Context, req *cmpb.SeatsRequest) (*cmpb.SeatsResponse, error) {
	datanodeAddr, exists := s.sessions.CheckAffinity(req.PassengerId)

	if exists {
		log.Printf("[Coordinador] Sesión encontrada para %s. Solicitando asientos DIRECTO a %s", req.FlightId, datanodeAddr)

		// A. CONEXIÓN DIRECTA (Ruta Rápida / RYW)
		dnClient, conn, err := s.getDatanodeClient(datanodeAddr)
		if err != nil {
			log.Printf("Error conectando directo al Datanode %s: %v", datanodeAddr, err)
			// Broker
			return s.brokerClient.GetSeats(ctx, req)
		}
		defer conn.Close()

		return dnClient.GetSeats(ctx, req)

	} else {
		log.Printf("[Coordinador] Sin sesión para %s. Solicitando asientos vía BROKER.", req.FlightId)
		// Broker
		return s.brokerClient.GetSeats(ctx, req)
	}
}