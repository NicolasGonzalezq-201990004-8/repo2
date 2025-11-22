package main

import (
    "context"
    "log"
    "time"

    cmpb "lab_3/proto/common/cmpb"
    dpb  "lab_3/proto/datanode/dpb"
)

func (s *DatanodeServer) sendGossip(update *cmpb.FlightUpdate) {
    s.mu.Lock()
    peers := append([]dpb.DatanodeServiceClient(nil), s.gossipClients...)
    peerAddrs := append([]string(nil), s.gossipAddrs...)
    s.mu.Unlock()

    for i, client := range peers {
        addr := peerAddrs[i]
        go func(cli dpb.DatanodeServiceClient, peer string) {
            _, err := cli.Gossip(context.Background(), update)
            if err != nil {
                log.Printf("[Datanode %s] Gossip error hacia %s: %v", s.id, peer, err)
            }
        }(client, addr)
    }
}

func (s *DatanodeServer) StartPeriodicGossip() {
    ticker := time.NewTicker(3 * time.Second)

    log.Printf("[Datanode %s] Gossip periódico activado.", s.id)

    for {
        <-ticker.C

        s.mu.Lock()
        for flightID, st := range s.flights {
            update := &cmpb.FlightUpdate{
                FlightId:    flightID,
                Status:      st.Status,
                Gate:        st.Gate,
                VectorClock: copyVectorClock(st.VectorClock),
            }
            go s.sendGossip(update)
        }
        s.mu.Unlock()
    }
}

func (s *DatanodeServer) Gossip(ctx context.Context, update *cmpb.FlightUpdate) (*cmpb.Empty, error) {

    s.applyUpdateLocal(update)
    return &cmpb.Empty{}, nil
}
