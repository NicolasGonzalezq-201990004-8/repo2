package main

import (
	"context"
	"log"
	"math/rand"
	"os"
	"strings"
	"sync"
	"time"

	cspb "lab_3/proto/consensus/cspb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/codes"
	"google.golang.org/grpc/credentials/insecure"
	"google.golang.org/grpc/status"
)

const (
	StateFollower  = "Follower"
	StateCandidate = "Candidate"
	StateLeader    = "Leader"
)

type ConsensusNode struct {
	mu    sync.Mutex
	ID    string
	State string

	CurrentTerm int32
	VotedFor    string
	Log         *PersistentLog

	Peers map[string]cspb.InternalConsensusServiceClient

	PeerAddresses    map[string]string
	AvailableRunways []string

	ElectionTimer *time.Timer
}

func NewConsensusNode() *ConsensusNode {
	nodeID := os.Getenv("NODE_ID")
	if nodeID == "" {
		log.Fatal("Variable de entorno NODE_ID no está configurada.")
	}

	peerAddrsStr := os.Getenv("PEER_ADDRESSES")
	if peerAddrsStr == "" {
		log.Fatal("Variable de entorno PEER_ADDRESSES no está configurada.")
	}

	node := &ConsensusNode{
		ID:               nodeID,
		State:            StateFollower, // Todos empiezan como seguidores
		CurrentTerm:      0,
		VotedFor:         "",
		Log:              NewPersistentLog(), // Inicializa el log (Paso 1)
		Peers:            make(map[string]cspb.InternalConsensusServiceClient),
		PeerAddresses:    make(map[string]string),
		AvailableRunways: []string{"Pista-1", "Pista-2", "Pista-3", "Pista-4", "Pista-5"},
	}

	peers := strings.Split(peerAddrsStr, ",")
	for _, peer := range peers {
		parts := strings.Split(peer, "=") // ej: "ATC-2", "localhost:50062"
		if len(parts) != 2 {
			log.Fatalf("Formato de PEER_ADDRESSES inválido: %s", peer)
		}
		peerID := parts[0]
		peerAddr := parts[1]

		node.PeerAddresses[peerID] = peerAddr
	}

	log.Printf("[%s] Nodo inicializado. Estado: %s", node.ID, node.State)
	return node
}

// Conexión con los demás nodos
func (n *ConsensusNode) ConnectToPeers() {
	for peerID, addr := range n.PeerAddresses {
		// Establecemos conexión insegura para el laboratorio
		conn, err := grpc.Dial(addr, grpc.WithTransportCredentials(insecure.NewCredentials()))
		if err != nil {
			log.Printf("[%s] Error crítico al intentar conectar con peer %s (%s): %v", n.ID, peerID, addr, err)
			continue
		}

		client := cspb.NewInternalConsensusServiceClient(conn)

		n.mu.Lock()
		n.Peers[peerID] = client
		n.mu.Unlock()

		log.Printf("[%s] Conectado exitosamente a peer %s en %s", n.ID, peerID, addr)
	}
}

func (n *ConsensusNode) findAvailableRunway() (string, bool) {

	occupied := make(map[string]bool)

	for i, entry := range n.Log.Entries {
		log.Printf("[DEBUG LOG REPLAY] Index: %d | Op: %s | Resource: %s", i, entry.Operation, entry.Resource)
		switch entry.Operation {
		case "ASSIGN":
			occupied[entry.Resource] = true
		case "RELEASE":
			delete(occupied, entry.Resource)
		default:
			log.Printf("[%s] ALERTA: Operación desconocida en log: '%s'", n.ID, entry.Operation)
		}
	}

	for _, runway := range n.AvailableRunways {
		if !occupied[runway] {
			return runway, true
		}
	}

	return "", false
}

func (n *ConsensusNode) replicateToQuorum(ctx context.Context, entry *cspb.Decision) bool {
	votes := 1

	clusterSize := len(n.Peers) + 1
	quorum := (clusterSize / 2) + 1

	ackChan := make(chan bool, len(n.Peers))

	for peerID, peerClient := range n.Peers {
		go func(pid string, client cspb.InternalConsensusServiceClient) {

			resp, err := client.AppendEntry(ctx, entry)

			if err == nil && resp.Committed {
				ackChan <- true // Voto positivo
			} else {
				log.Printf("[%s] Falló replicación a %s: %v", n.ID, pid, err)
				ackChan <- false
			}
		}(peerID, peerClient)
	}

	// Esperamos respuestas
	for i := 0; i < len(n.Peers); i++ {
		success := <-ackChan
		if success {
			votes++
		}
		// Si ya tenemos quórum, no necesitamos esperar al resto
		if votes >= quorum {
			return true
		}
	}

	return votes >= quorum
}

func (n *ConsensusNode) HandleRunwayAssignment(ctx context.Context, req *cspb.RunwayRequest) (*cspb.RunwayResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// VERIFICACIÓN DE LÍDER
	if n.State != StateLeader {
		// Devolvemos error para que el Broker aplique su lógica de reintento
		return nil, status.Errorf(codes.FailedPrecondition, "Nodo %s no es líder", n.ID)
	}

	var runwayID string
	switch req.Operation {
	case "ASSIGN":
		var available bool
		runwayID, available = n.findAvailableRunway()
		if !available {
			return &cspb.RunwayResponse{Success: false}, status.Errorf(codes.ResourceExhausted, "No hay pistas")
		}
	case "RELEASE":
		runwayID = req.RunwayId
	default:
		log.Printf("[%s] ALERTA: Operación desconocida en el request: '%s'", n.ID, req.Operation)
	}

	log.Printf("[%s] LÍDER recibió solicitud de pista para vuelo %s", n.ID, req.FlightId)

	prevIndex := n.Log.GetLastIndex()
	prevTerm := n.Log.GetLastTerm()

	entry := &cspb.Decision{
		Resource:   runwayID,
		AssignedTo: req.FlightId,
		Operation:  req.Operation,
		PrevTerm:   prevTerm,  // para consistencia
		PrevIndex:  prevIndex, // para consistencia
	}

	n.Log.Append(entry)

	success := n.replicateToQuorum(ctx, entry)

	if success {
		log.Printf("[%s] QUÓRUM ALCANZADO. Asignación %s -> %s Comprometida.", n.ID, runwayID, req.FlightId)
		return &cspb.RunwayResponse{
			Success:  true,
			RunwayId: runwayID,
		}, nil
	}

	log.Printf("[%s] FALLÓ EL QUÓRUM para %s", n.ID, req.FlightId)
	return nil, status.Errorf(codes.Aborted, "No se logró consenso (fallo de quórum)")
}

func (n *ConsensusNode) HandleAppendEntry(ctx context.Context, req *cspb.Decision) (*cspb.DecisionAck, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// REINICIAR ELECTION TIMER (Heartbeat)
	if n.ElectionTimer != nil {
		n.ElectionTimer.Stop()
		// Reiniciamos con una duración aleatoria (definiremos resetElectionTimer después)
		n.resetElectionTimer()
	}

	// Si éramos Candidatos y recibimos esto, significa que alguien más ganó. Volvemos a ser Seguidores.
	if n.State == StateCandidate {
		n.State = StateFollower
		log.Printf("[%s] Líder detectado. Volviendo a estado Follower.", n.ID)
	}

	// VERIFICACIÓN DE CONSISTENCIA

	lastLogIndex := n.Log.GetLastIndex()

	// Caso El líder está más avanzado que nosotros (Hueco en el log)
	if req.PrevIndex > lastLogIndex {
		log.Printf("[%s] Conflicto de Log detectado. Líder dice PrevIndex=%d, yo tengo %d. RECHAZANDO.",
			n.ID, req.PrevIndex, lastLogIndex)

		return &cspb.DecisionAck{
			Committed: false,
			Msg:       "Log inconsistente (Missing entries)",
		}, nil
	}

	// Caso Verificar si el término en PrevIndex coincide.

	// ESCRIBIR EN EL LOG
	if req.Resource == "" && req.AssignedTo == "" {
		return &cspb.DecisionAck{Committed: true}, nil
	}

	n.Log.Append(req)

	log.Printf("[%s] Entrada replicada exitosamente: %s -> %s", n.ID, req.Resource, req.AssignedTo)

	return &cspb.DecisionAck{Committed: true}, nil
}

func (n *ConsensusNode) HandleRequestVote(ctx context.Context, req *cspb.VoteRequest) (*cspb.VoteResponse, error) {
	n.mu.Lock()
	defer n.mu.Unlock()

	// Si el término del solicitante es mayor, actualizamos el nuestro.
	if req.Term > n.CurrentTerm {
		log.Printf("[%s] Término mayor detectado (%d > %d). Volviendo a Follower.", n.ID, req.Term, n.CurrentTerm)
		n.CurrentTerm = req.Term
		n.State = StateFollower
		n.VotedFor = "" // Reseteamos el voto porque es un nuevo término
	}

	// RECHAZAR SI EL TÉRMINO ES ANTIGUO
	if req.Term < n.CurrentTerm {
		// El candidato está desactualizado. No votamos por él.
		return &cspb.VoteResponse{
			VoteGranted: false,
			Term:        n.CurrentTerm,
		}, nil
	}

	// VERIFICAR VOTO
	if n.VotedFor == "" || n.VotedFor == req.CandidateId {

		n.VotedFor = req.CandidateId
		n.State = StateFollower // Nos mantenemos como seguidores

		// Reiniciar el ElectionTimer porque hemos escuchado de un candidato válido
		if n.ElectionTimer != nil {
			n.ElectionTimer.Stop()
			n.resetElectionTimer()
		}

		log.Printf("[%s] Voto otorgado a %s para el término %d.", n.ID, req.CandidateId, n.CurrentTerm)

		return &cspb.VoteResponse{
			VoteGranted: true,
			Term:        n.CurrentTerm,
		}, nil
	}

	// RECHAZO POR DEFECTO
	return &cspb.VoteResponse{
		VoteGranted: false,
		Term:        n.CurrentTerm,
	}, nil
}

func (n *ConsensusNode) resetElectionTimer() {
	duration := time.Duration(3000+rand.Intn(3000)) * time.Millisecond
	n.ElectionTimer.Reset(duration)
}

func (n *ConsensusNode) StartElectionTimer() {
	n.mu.Lock()
	n.ElectionTimer = time.NewTimer(10 * time.Hour)
	n.resetElectionTimer()
	n.mu.Unlock()

	for {

		<-n.ElectionTimer.C

		n.mu.Lock()
		if n.State != StateLeader {
			log.Printf("[%s] ¡Election Timeout! Iniciando elección...", n.ID)
			n.startElection()
		}
		n.mu.Unlock()
	}
}

func (n *ConsensusNode) startElection() {
	n.State = StateCandidate
	n.CurrentTerm++
	n.VotedFor = n.ID
	n.resetElectionTimer()

	log.Printf("[%s] Convirtiéndose en CANDIDATO (Término %d). Solicitando votos...", n.ID, n.CurrentTerm)

	votesReceived := 1

	for peerID, client := range n.Peers {
		go func(pid string, c cspb.InternalConsensusServiceClient, term int32) {

			req := &cspb.VoteRequest{
				Term:        term,
				CandidateId: n.ID,
			}

			ctx, cancel := context.WithTimeout(context.Background(), 500*time.Millisecond)
			defer cancel()

			resp, err := c.RequestVote(ctx, req)

			n.mu.Lock()
			defer n.mu.Unlock()

			if n.State != StateCandidate || n.CurrentTerm != term {
				return
			}

			if err == nil {
				if resp.VoteGranted {
					votesReceived++
					log.Printf("[%s] Voto recibido de %s. Total: %d", n.ID, pid, votesReceived)

					clusterSize := len(n.Peers) + 1
					quorum := (clusterSize / 2) + 1

					if votesReceived >= quorum {
						n.becomeLeader() // Ganamos!
					}
				} else if resp.Term > n.CurrentTerm {

					n.State = StateFollower
					n.CurrentTerm = resp.Term
					n.VotedFor = ""
				}
			}
		}(peerID, client, n.CurrentTerm)
	}
}

func (n *ConsensusNode) becomeLeader() {
	if n.State == StateLeader {
		return
	}
	n.State = StateLeader
	log.Printf("[%s] ¡QUÓRUM ALCANZADO! Ascendiendo a LÍDER del término %d.", n.ID, n.CurrentTerm)

	// Iniciar envío periódico de Heartbeats (AppendEntry vacío)
	go n.startHeartbeatLoop()
}

func (n *ConsensusNode) startHeartbeatLoop() {
	ticker := time.NewTicker(1 * time.Second)
	defer ticker.Stop()

	for {

		n.mu.Lock()
		if n.State != StateLeader {
			n.mu.Unlock()
			return
		}
		term := n.CurrentTerm
		n.mu.Unlock()

		// Enviar Heartbeat a todos
		lastIndex := n.Log.GetLastIndex()

		hb := &cspb.Decision{
			PrevTerm:  term,
			PrevIndex: lastIndex,
		}

		n.replicateToQuorum(context.Background(), hb)

		<-ticker.C
	}
}