package main

import (
	"context"
	"log"
	"os"
	"sync"
	"time"

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

	brokerAddr   string
	brokerConn   *grpc.ClientConn
	brokerClient bpb.BrokerServiceClient
	brokerMu     sync.RWMutex
}

func NewCoordinatorServer() *CoordinatorServer {
	brokerAddr := os.Getenv("BROKER_ADDR")
	if brokerAddr == "" {
		brokerAddr = "broker:50050"
		log.Printf("[WARN] BROKER_ADDR no configurada, usando default local: %s", brokerAddr)
	}

	s := &CoordinatorServer{
		sessions:   NewSessionMap(),
		brokerAddr: brokerAddr,
	}

	go s.startBrokerDialLoop()

	return s
}

func (s *CoordinatorServer) getDatanodeClient(addr string) (dpb.DatanodeServiceClient, *grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, addr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		return nil, nil, err
	}
	return dpb.NewDatanodeServiceClient(conn), conn, nil
}

func (s *CoordinatorServer) startBrokerDialLoop() {
	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		conn, err := grpc.DialContext(ctx, s.brokerAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		cancel()

		if err == nil {
			s.brokerMu.Lock()
			s.brokerConn = conn
			s.brokerClient = bpb.NewBrokerServiceClient(conn)
			s.brokerMu.Unlock()

			log.Printf("[Coordinator] Conectado al Broker en %s", s.brokerAddr)
			return
		}

		log.Printf("[Coordinator] Reintento %d conectando al Broker en %s: %v", attempt, s.brokerAddr, err)
		time.Sleep(2 * time.Second)
	}
}

func (s *CoordinatorServer) waitForBrokerClient(ctx context.Context) (bpb.BrokerServiceClient, error) {
	for {
		s.brokerMu.RLock()
		client := s.brokerClient
		s.brokerMu.RUnlock()

		if client != nil {
			return client, nil
		}

		select {
		case <-ctx.Done():
			return nil, ctx.Err()
		case <-time.After(500 * time.Millisecond):
		}
	}
}

func (s *CoordinatorServer) CheckIn(ctx context.Context, req *cmpb.CheckInRequest) (*cmpb.CheckInResponse, error) {
	log.Printf("[Coordinator] Iniciando transacción RYW para Vuelo %s", req.FlightId)

	brokerClient, err := s.waitForBrokerClient(ctx)
	if err != nil {
		log.Printf("[Coordinator] Broker no disponible: %v", err)
		return nil, err
	}

	dnResp, err := brokerClient.GetDatanode(ctx, &cmpb.Empty{})
	if err != nil {
		log.Printf("Error obteniendo Datanode del Broker: %v", err)
		return nil, err
	}
	targetAddr := dnResp.DatanodeId

	clientID := req.PassengerId
	s.sessions.RegisterAffinity(clientID, targetAddr)

	log.Printf("[Coordinator] Afinidad fijada: Cliente -> %s", targetAddr)

	req.TargetDatanodeId = targetAddr

	resp, err := brokerClient.CheckIn(ctx, req)
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

			brokerClient, err := s.waitForBrokerClient(ctx)
			if err != nil {
				log.Printf("[Coordinator] Broker no disponible: %v", err)
				return nil, err
			}

			return brokerClient.GetBoardingPass(ctx, req)
		}
		defer conn.Close()

		return dnClient.GetBoardingPass(ctx, req)

	}

	log.Printf("[Coordinador] Sin sesión para %s. Redirigiendo al Broker.", req.FlightId)

	brokerClient, err := s.waitForBrokerClient(ctx)
	if err != nil {
		log.Printf("[Coordinator] Broker no disponible: %v", err)
		return nil, err
	}

	return brokerClient.GetBoardingPass(ctx, req)
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
			brokerClient, err := s.waitForBrokerClient(ctx)
			if err != nil {
				log.Printf("[Coordinator] Broker no disponible: %v", err)
				return nil, err
			}

			return brokerClient.GetSeats(ctx, req)
		}
		defer conn.Close()

		return dnClient.GetSeats(ctx, req)

	}

	log.Printf("[Coordinador] Sin sesión para %s. Solicitando asientos vía BROKER.", req.FlightId)

	brokerClient, err := s.waitForBrokerClient(ctx)
	if err != nil {
		log.Printf("[Coordinator] Broker no disponible: %v", err)
		return nil, err
	}

	return brokerClient.GetSeats(ctx, req)
}
