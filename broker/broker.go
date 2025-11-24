package main

import (
	"fmt"
	"log"
	"os"
	"strings"
	"sync"
	"time"

	"context"

	bpb "lab_3/proto/broker/bpb"
	cspb "lab_3/proto/consensus/cspb"

	// cdpb "lab_3/proto/coordinator/cdpb"
	cmpb "lab_3/proto/common/cmpb"
	dpb "lab_3/proto/datanode/dpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

type BrokerServer struct {
	bpb.UnimplementedBrokerServiceServer

	datanodeClients  []dpb.DatanodeServiceClient
	consensusClients []cspb.ConsensusServiceClient

	datanodeAddresses  []string
	consensusAddresses []string

	mu sync.Mutex

	datanodeIndex   int
	leaderHintIndex int

	brokerID string

	flightCounters map[string]int32
	reportChan     chan string
}

const (
	ReportFilename = "Reporte.txt"
)

func NewBrokerServer() *BrokerServer {
	datanodeAddrsStr := os.Getenv("DATANODE_ADDRESSES")
	if datanodeAddrsStr == "" {
		log.Println("[WARN] DATANODE_ADDRESSES no configurada. Usando layout local predeterminado.")
		datanodeAddrsStr = "datanode1:50051,datanode2:50051,datanode3:50051"
	}
	datanodeAddrs := splitAndTrim(datanodeAddrsStr)

	consensusAddrsStr := os.Getenv("CONSENSUS_ADDRESSES")
	if consensusAddrsStr == "" {
		log.Println("[WARN] CONSENSUS_ADDRESSES no configurada. Usando layout local predeterminado.")
		consensusAddrsStr = "consensus1:50060,consensus2:50060,consensus3:50060"
	}
	consensusAddrs := splitAndTrim(consensusAddrsStr)

	server := &BrokerServer{
		datanodeClients:    make([]dpb.DatanodeServiceClient, 0),
		consensusClients:   make([]cspb.ConsensusServiceClient, 0),
		datanodeAddresses:  make([]string, 0),
		consensusAddresses: make([]string, 0),
		datanodeIndex:      0,
		leaderHintIndex:    0,

		brokerID:       "broker_main",
		flightCounters: make(map[string]int32),
		reportChan:     make(chan string, 100),
	}
	for _, addr := range datanodeAddrs {
		log.Printf("Intentando conexión con el Datanode %s", addr)
		if err := server.connectDatanode(addr); err != nil {
			log.Printf("[WARN] No se pudo conectar al Datanode %s en el arranque: %v. Se reintentará en segundo plano.", addr, err)
			go server.retryConnectDatanode(addr)
		}
	}

	for _, addr := range consensusAddrs {
		log.Printf("Intentando conexión con el Nodo de consenso %s", addr)
		if err := server.connectConsensus(addr); err != nil {
			log.Printf("[WARN] No se pudo conectar al Nodo de Consenso %s en el arranque: %v. Se reintentará en segundo plano.", addr, err)
			go server.retryConnectConsensus(addr)
		}
	}
	//FASE 1
	server.loadFlightCatalog()

	return server
}

func (s *BrokerServer) QueryFlight(ctx context.Context, req *cmpb.GetFlightRequest) (*cmpb.FlightState, error) {
	log.Printf("Broker recibió QueryFlight para: %s", req.FlightId)

	datanodeClient, _ := s.selectNextDatanode()
	if datanodeClient == nil {
		return nil, status.Errorf(codes.Unavailable, "No hay datanodes disponibles")
	}

	log.Printf("Redirigiendo consulta de vuelo al Datanode (índice %d)", s.datanodeIndex)

	resp, err := datanodeClient.GetFlightState(ctx, req)

	if err != nil {
		log.Printf("Error al recibir respuesta del Datanode: %v", err)
		return nil, err
	}

	return resp, nil
}

func (s *BrokerServer) GetDatanode(ctx context.Context, req *cmpb.Empty) (*bpb.DatanodeResponse, error) {
	datanodeClient, selectedDatanodeAddr := s.selectNextDatanode()
	if datanodeClient == nil {
		return nil, status.Errorf(codes.Unavailable, "No hay Datanodes disponibles para afinidad.")
	}

	log.Printf("[Broker] Asignando Datanode %s al Coordinador para afinidad.", selectedDatanodeAddr)

	return &bpb.DatanodeResponse{
		DatanodeId: selectedDatanodeAddr,
	}, nil
}

func (s *BrokerServer) CheckIn(ctx context.Context, req *cmpb.CheckInRequest) (*cmpb.CheckInResponse, error) {
	var datanodeClient dpb.DatanodeServiceClient

	// 1) Intentar respetar afinidad si viene TargetDatanodeId
	if req.TargetDatanodeId != "" {
		s.mu.Lock()
		for i, addr := range s.datanodeAddresses {
			if addr == req.TargetDatanodeId {
				datanodeClient = s.datanodeClients[i]
				break
			}
		}
		s.mu.Unlock()

		if datanodeClient != nil {
			log.Printf("[Broker] Check-In con afinidad: redirigiendo a %s", req.TargetDatanodeId)
		} else {
			log.Printf("[Broker] WARNING: TargetDatanodeId=%s no encontrado, usando round-robin", req.TargetDatanodeId)
		}
	}

	// 2) Si no había afinidad válida, usar round-robin
	if datanodeClient == nil {
		var addr string
		datanodeClient, addr = s.selectNextDatanode()
		if datanodeClient == nil {
			return nil, status.Errorf(codes.Unavailable, "No hay Datanodes disponibles para el Check-in.")
		}
		log.Printf("[Broker] Check-In sin afinidad. Usando Datanode %s (round-robin).", addr)
	}

	log.Printf("[Broker] Enrutando solicitud de Check-In para Vuelo %s (Passenger=%s, Seat=%s).",
		req.FlightId, req.PassengerId, req.Seat)

	resp, err := datanodeClient.CheckIn(ctx, req)
	if err != nil {
		log.Printf("Error al redirigir Check-In al Datanode: %v", err)
		return nil, err
	}

	if resp.Success {
		log.Printf("[Broker] Check-In exitoso para Vuelo %s | Asiento: %s.", req.FlightId, req.Seat)
		s.reportChan <- fmt.Sprintf("CHECK-IN EXITOSO: Vuelo %s | Asiento: %s (Passenger=%s)",
			req.FlightId, req.Seat, req.PassengerId)
	} else {
		log.Printf("[Broker] Check-In fallido para Vuelo %s | Asiento: %s.", req.FlightId, req.Seat)
		s.reportChan <- fmt.Sprintf("CHECK-IN FALLIDO: Vuelo %s | Asiento: %s (Passenger=%s)",
			req.FlightId, req.Seat, req.PassengerId)
	}

	return resp, nil
}

func (s *BrokerServer) RequestRunwayAssignment(ctx context.Context, flightID string) (*cspb.RunwayResponse, error) {
	if len(s.consensusClients) == 0 {
		return nil, status.Errorf(codes.Unavailable, "Subsistema de Consenso no disponible.")
	}

	assignReq := &cspb.RunwayRequest{
		FlightId:  flightID,
		Operation: "ASSIGN",
	}

	log.Printf("[Broker] Solicitando pista para Vuelo %s", flightID)
	resp, err := s.sendConsensusRequest(ctx, assignReq)
	if err != nil {
		log.Printf("[Broker] Falló la asignación de pista para %s: %v", flightID, err)
		return nil, err
	}

	s.reportChan <- fmt.Sprintf("ASIGNACIÓN EXITOSA: Vuelo %s | Pista %s", flightID, resp.RunwayId)

	go func(rId, fId string) {
		time.Sleep(5 * time.Second) //TIEMPO DE OCUPACIÓN DE PISTA

		log.Printf("[Broker] Vuelo %s liberando pista %s...", fId, rId)

		releaseReq := &cspb.RunwayRequest{
			FlightId:  fId,
			RunwayId:  rId,
			Operation: "RELEASE",
		}

		_, err := s.sendConsensusRequest(context.Background(), releaseReq)

		if err != nil {
			log.Printf("[Broker] Error al liberar pistsa %s: %v", rId, err)
		} else {
			log.Printf("[Broker] PISTA LIBERADA: %s ahora está libre.", rId)
			s.reportChan <- fmt.Sprintf("LIBERACIÓN: Pista %s (Vuelo %s)", rId, fId)
		}
	}(resp.RunwayId, flightID)

	return resp, nil
}

func (s *BrokerServer) BroadcastUpdate(ctx context.Context, update *cmpb.FlightUpdate) {
	dnClient, addr := s.selectNextDatanode()
	if dnClient == nil {
		log.Printf("[Broker] No hay datanodes disponibles para BroadcastUpdate de vuelo %s", update.FlightId)
		return
	}

	log.Printf("[Broker] Enviando actualización de vuelo %s (Status: %s, Gate: %s) al Datanode %s",
		update.FlightId, update.Status, update.Gate, addr)

	go func() {
		_, err := dnClient.ApplyUpdate(context.Background(), update)
		if err != nil {
			log.Printf("[Broker] Error al enviar actualización (Vuelo: %s) al Datanode %s: %v",
				update.FlightId, addr, err)
		}
	}()
}

func (s *BrokerServer) GetSeats(ctx context.Context, req *cmpb.SeatsRequest) (*cmpb.SeatsResponse, error) {
	log.Printf("[Broker] Recibido GetSeats para Vuelo %s", req.FlightId)
	datanodeClient, _ := s.selectNextDatanode()
	if datanodeClient == nil {
		return nil, status.Errorf(codes.Unavailable, "No hay datanodes disponibles")
	}
	resp, err := datanodeClient.GetSeats(ctx, req)

	if err != nil {
		log.Printf("Error al recibir respuesta del Datanode: %v", err)
		return nil, err
	}

	return resp, nil
}

func (s *BrokerServer) GetBoardingPass(ctx context.Context, req *cmpb.BoardingPassRequest) (*cmpb.BoardingPassResponse, error) {
	log.Printf("[Broker] Recibido GetBoardingPass para Vuelo %s", req.FlightId)
	datanodeClient, _ := s.selectNextDatanode()
	if datanodeClient == nil {
		return nil, status.Errorf(codes.Unavailable, "No hay datanodes disponibles")
	}
	resp, err := datanodeClient.GetBoardingPass(ctx, req)

	if err != nil {
		log.Printf("Error al recibir respuesta del Datanode: %v", err)
		return nil, err
	}

	return resp, nil
}

func (s *BrokerServer) sendConsensusRequest(ctx context.Context, req *cspb.RunwayRequest) (*cspb.RunwayResponse, error) {
	if len(s.consensusClients) == 0 {
		return nil, status.Errorf(codes.Unavailable, "Subsistema de Consenso no disponible.")
	}

	maxRetries := len(s.consensusClients) + 1

	for attempt := 0; attempt < maxRetries; attempt++ {
		s.mu.Lock()
		clientIndex := s.leaderHintIndex
		client := s.consensusClients[clientIndex]
		s.mu.Unlock()

		log.Printf("[Broker] Enviando operación '%s' para Vuelo %s al Nodo %d...", req.Operation, req.FlightId, clientIndex)

		resp, err := client.RequestRunwayAssignment(ctx, req)

		if err == nil {
			return resp, nil
		}

		st, ok := status.FromError(err)
		if ok && (st.Code() == codes.Unavailable || st.Code() == codes.FailedPrecondition) {
			log.Printf("[Broker] Nodo %d falló o no es líder. Reintentando...", clientIndex)

			s.mu.Lock()
			s.leaderHintIndex = (s.leaderHintIndex + 1) % len(s.consensusClients)
			s.mu.Unlock()

			time.Sleep(200 * time.Millisecond)
			continue
		}

		return nil, err
	}
	return nil, status.Errorf(codes.Unavailable, "No se pudo completar la operación de consenso después de %d reintentos", maxRetries)
}

func splitAndTrim(addressList string) []string {
	parts := strings.Split(addressList, ",")
	for i, p := range parts {
		parts[i] = strings.TrimSpace(p)
	}
	return parts
}

func (s *BrokerServer) connectDatanode(addr string) error {
	conn, err := dialWithTimeout(addr)
	if err != nil {
		return err
	}

	client := dpb.NewDatanodeServiceClient(conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.datanodeClients = append(s.datanodeClients, client)
	s.datanodeAddresses = append(s.datanodeAddresses, addr)
	log.Printf("Broker conectado al Datanode %s", addr)
	return nil
}

func (s *BrokerServer) retryConnectDatanode(addr string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Printf("[Retry] Reintentando conexión con Datanode %s...", addr)
		if err := s.connectDatanode(addr); err != nil {
			log.Printf("[Retry] Datanode %s aún no disponible: %v", addr, err)
			continue
		}
		return
	}
}

func (s *BrokerServer) connectConsensus(addr string) error {
	conn, err := dialWithTimeout(addr)
	if err != nil {
		return err
	}

	client := cspb.NewConsensusServiceClient(conn)

	s.mu.Lock()
	defer s.mu.Unlock()

	s.consensusClients = append(s.consensusClients, client)
	s.consensusAddresses = append(s.consensusAddresses, addr)
	log.Printf("Broker conectado al nodo de Consenso %s", addr)
	return nil
}

func (s *BrokerServer) retryConnectConsensus(addr string) {
	ticker := time.NewTicker(5 * time.Second)
	defer ticker.Stop()

	for {
		<-ticker.C
		log.Printf("[Retry] Reintentando conexión con Nodo de Consenso %s...", addr)
		if err := s.connectConsensus(addr); err != nil {
			log.Printf("[Retry] Nodo de Consenso %s aún no disponible: %v", addr, err)
			continue
		}
		return
	}
}

func dialWithTimeout(addr string) (*grpc.ClientConn, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	return grpc.DialContext(ctx, addr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
}
