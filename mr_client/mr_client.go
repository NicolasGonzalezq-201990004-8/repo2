package main

import (
	"context"
	"fmt"
	"log"
	"time"

	bpb "lab_3/proto/broker/bpb"
	cmpb "lab_3/proto/common/cmpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

type MRClient struct {
	brokerAddr    string
	conn          *grpc.ClientConn
	client        bpb.BrokerServiceClient
	localVersions map[string]map[string]int32 // flight → vector clock
}

func NewMRClient(brokerAddr string) (*MRClient, error) {
	ctx, cancel := context.WithTimeout(context.Background(), 8*time.Second)
	defer cancel()

	conn, err := grpc.DialContext(ctx, brokerAddr,
		grpc.WithTransportCredentials(insecure.NewCredentials()),
		grpc.WithBlock(),
	)
	if err != nil {
		return nil, fmt.Errorf("no se pudo conectar al broker: %v", err)
	}

	c := &MRClient{
		brokerAddr:    brokerAddr,
		conn:          conn,
		client:        bpb.NewBrokerServiceClient(conn),
		localVersions: make(map[string]map[string]int32),
	}

	log.Printf("[MR] Cliente MR conectado a Broker %s", brokerAddr)
	return c, nil
}

func dominates(vc1, vc2 map[string]int32) bool {
	for node, v2 := range vc2 {
		if vc1[node] < v2 {
			return false
		}
	}
	return true
}

func cloneVC(vc map[string]int32) map[string]int32 {
	out := make(map[string]int32)
	for k, v := range vc {
		out[k] = v
	}
	return out
}

func (c *MRClient) QueryFlightMR(flightID string) (*cmpb.FlightState, error) {

	localVC := c.localVersions[flightID]
	if localVC == nil {
		localVC = make(map[string]int32)
	}

	req := &cmpb.GetFlightRequest{
		FlightId:      flightID,
		ClientVersion: cloneVC(localVC),
	}

	log.Printf("[MR] Enviando QueryFlight(%s) con VC local=%v", flightID, localVC)

	ctx, cancel := context.WithTimeout(context.Background(), 4*time.Second)
	defer cancel()

	resp, err := c.client.QueryFlight(ctx, req)
	if err != nil {
		return nil, fmt.Errorf("[MR] Error consultando vuelo %s: %v", flightID, err)
	}

	serverVC := resp.VectorClock
	log.Printf("[MR] Broker respondió VC=%v | Estado=%s | Gate=%s",
		serverVC, resp.Status, resp.Gate)

	// MONOTONIC READS
	if dominates(serverVC, localVC) {
		c.localVersions[flightID] = cloneVC(serverVC)
		log.Printf("[MR] Actualización aceptada. Nuevo VC local=%v", serverVC)
	} else {
		log.Printf("[MR] Rechazo de actualización: servidor devolvió VC antiguo.")
		resp.VectorClock = cloneVC(localVC)
	}

	return resp, nil
}
