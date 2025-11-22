package main

import (
	"context"

	cspb "lab_3/proto/consensus/cspb"
)

type ConsensusServer struct {
	cspb.UnimplementedConsensusServiceServer
	cspb.UnimplementedInternalConsensusServiceServer

	Node *ConsensusNode
}

func NewConsensusServer(node *ConsensusNode) *ConsensusServer {
	return &ConsensusServer{
		Node: node,
	}
}

// --- SERVICIO PÚBLICO  ---

// RequestRunwayAssignment recibe la solicitud del Broker para asignar una pista.
func (s *ConsensusServer) RequestRunwayAssignment(ctx context.Context, req *cspb.RunwayRequest) (*cspb.RunwayResponse, error) {
	// Delegamos la lógica al nodo Raft
	return s.Node.HandleRunwayAssignment(ctx, req)
}

// --- SERVICIO INTERNO (Entre Nodos de Consenso) ---

// RequestVote es llamado por un Candidato para solicitar el voto de este nodo.
func (s *ConsensusServer) RequestVote(ctx context.Context, req *cspb.VoteRequest) (*cspb.VoteResponse, error) {
	return s.Node.HandleRequestVote(ctx, req)
}

// AppendEntry es llamado por el Líder para replicar logs o enviar heartbeats.
func (s *ConsensusServer) AppendEntry(ctx context.Context, req *cspb.Decision) (*cspb.DecisionAck, error) {
	return s.Node.HandleAppendEntry(ctx, req)
}