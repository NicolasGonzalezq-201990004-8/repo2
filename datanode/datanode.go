package main

import (
    "context"
    cmpb "lab_3/proto/common/cmpb"
    dpb "lab_3/proto/datanode/dpb"
    "log"
    "os"
    "strings"
    "sync"

    "google.golang.org/grpc"
    "google.golang.org/grpc/codes"
    "google.golang.org/grpc/status"
)

type DatanodeServer struct {
    dpb.UnimplementedDatanodeServiceServer

    id string
    mu sync.Mutex

    flights map[string]*cmpb.FlightState
    seats   map[string]map[string]bool

	boarding map[string]map[string]string

    gossipClients []dpb.DatanodeServiceClient
    gossipAddrs   []string
}

func NewDatanodeServer() *DatanodeServer {
    id := os.Getenv("DATANODE_ID")
    if id == "" {
        id = "datanode_default"
    }

    myAddr := os.Getenv("DATANODE_ADDR")
    if myAddr == "" {
        log.Fatal("Falta DATANODE_ADDR")
    }

    allAddrsStr := os.Getenv("DATANODE_ADDRESSES")
    if allAddrsStr == "" {
        log.Fatal("Falta DATANODE_ADDRESSES")
    }
    allAddrs := strings.Split(allAddrsStr, ",")

    s := &DatanodeServer{
		id:            id,
		flights:       make(map[string]*cmpb.FlightState),
		seats:         make(map[string]map[string]bool),
		boarding:      make(map[string]map[string]string),
		gossipClients: make([]dpb.DatanodeServiceClient, 0),
		gossipAddrs:   make([]string, 0),
	}

    for _, addr := range allAddrs {
        addr = strings.TrimSpace(addr)
        if addr == myAddr {
            continue
        }
        conn, err := grpc.Dial(addr, grpc.WithInsecure())
        if err != nil {
            log.Printf("[Datanode %s] No se pudo conectar a peer %s", id, addr)
            continue
        }
        s.gossipClients = append(s.gossipClients, dpb.NewDatanodeServiceClient(conn))
        s.gossipAddrs = append(s.gossipAddrs, addr)
    }

    return s
}

func priority(state string) int {
    switch state {
    case "Cancelado":
        return 5
    case "En vuelo":
        return 4
    case "Retrasado":
        return 3
    case "Embarcado":
        return 2
    case "A tiempo":
        return 1
    }
    return 0
}

func concurrent(vc1, vc2 map[string]int32) bool {
    greater := false
    less := false

    for k := range vc1 {
        v1 := vc1[k]
        v2 := vc2[k]

        if v1 > v2 {
            greater = true
        }
        if v1 < v2 {
            less = true
        }
    }

    return greater && less
}

func compareVC(vc1, vc2 map[string]int32) int {
    greater := false
    less := false

    for k := range vc1 {
        if vc1[k] > vc2[k] {
            greater = true
        }
        if vc1[k] < vc2[k] {
            less = true
        }
    }

    if greater && !less {
        return 1
    }
    if less && !greater {
        return -1
    }
    return 0
}

func (s *DatanodeServer) applyUpdateLocal(update *cmpb.FlightUpdate) {
    s.mu.Lock()
    defer s.mu.Unlock()

    fs, ok := s.flights[update.FlightId]
    if !ok {
        fs = &cmpb.FlightState{
            FlightId:    update.FlightId,
            Status:      "",
            Gate:        "",
            VectorClock: make(map[string]int32),
        }
    }


    conflict := concurrent(fs.VectorClock, update.VectorClock)

    mergeVectorClock(fs.VectorClock, update.VectorClock)

    if conflict {

        log.Printf("[Datanode %s] CONFLICTO detectado vuelo=%s (local=%s remote=%s)",
            s.id, update.FlightId, fs.Status, update.Status)

        localP := priority(fs.Status)
        remoteP := priority(update.Status)

        if remoteP > localP {
            fs.Status = update.Status
            fs.Gate = update.Gate

        } else if remoteP == localP {

            vcCmp := compareVC(update.VectorClock, fs.VectorClock)

            if vcCmp > 0 {
                fs.Status = update.Status
                fs.Gate = update.Gate
            } else if vcCmp == 0 {
                if update.VectorClock[s.id] > fs.VectorClock[s.id] {
                    fs.Status = update.Status
                    fs.Gate = update.Gate
                }
            }
        }

    } else {
        if update.Status != "~NULL~" && update.Status != "" {
            fs.Status = update.Status
        }
        if update.Gate != "~NULL~" && update.Gate != "" {
            fs.Gate = update.Gate
        }
    }

    s.flights[update.FlightId] = fs
}

func (s *DatanodeServer) GetSeats(ctx context.Context, req *cmpb.SeatsRequest) (*cmpb.SeatsResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    seatMap, ok := s.seats[req.FlightId]
    if !ok {
        seatMap = make(map[string]bool)
        s.seats[req.FlightId] = seatMap
    }

    out := make(map[string]bool, len(seatMap))
    for seat, taken := range seatMap {
        out[seat] = taken
    }

    return &cmpb.SeatsResponse{
        SeatsMap: out,
    }, nil
}

func (s *DatanodeServer) CheckIn(ctx context.Context, req *cmpb.CheckInRequest) (*cmpb.CheckInResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    flightID := req.FlightId
    seat := req.Seat
    passenger := req.PassengerId

    seatMap, ok := s.seats[flightID]
    if !ok {
        seatMap = make(map[string]bool)
        s.seats[flightID] = seatMap
    }

    if seatMap[seat] {
        return &cmpb.CheckInResponse{Success: false}, nil
    }

    seatMap[seat] = true

    bpMap, ok := s.boarding[flightID]
    if !ok {
        bpMap = make(map[string]string)
        s.boarding[flightID] = bpMap
    }
    bpMap[passenger] = seat

    log.Printf("[Datanode %s] Check-In exitoso: Vuelo=%s Passenger=%s Seat=%s",
        s.id, flightID, passenger, seat)

    return &cmpb.CheckInResponse{Success: true}, nil
}

func (s *DatanodeServer) GetBoardingPass(ctx context.Context, req *cmpb.BoardingPassRequest) (*cmpb.BoardingPassResponse, error) {
    s.mu.Lock()
    defer s.mu.Unlock()

    flightID := req.FlightId
    passenger := req.PassengerId

    bpMap, ok := s.boarding[flightID]
    if !ok {
        log.Printf("[Datanode %s] BoardingPass: vuelo %s sin registros", s.id, flightID)
        return &cmpb.BoardingPassResponse{}, nil
    }

    seat, ok := bpMap[passenger]
    if !ok {
        log.Printf("[Datanode %s] BoardingPass: pasajero %s no tiene asiento en vuelo %s",
            s.id, passenger, flightID)
        return &cmpb.BoardingPassResponse{}, nil
    }

    gate := ""
    if fs, ok := s.flights[flightID]; ok {
        gate = fs.Gate
    }

    resp := &cmpb.BoardingPassResponse{
        FlightId: flightID,
        Seat:     seat,
        Gate:     gate,
    }

    log.Printf("[Datanode %s] BoardingPass generado: Vuelo=%s Passenger=%s Seat=%s Gate=%s",
        s.id, flightID, passenger, seat, gate)

    return resp, nil
}

func (s *DatanodeServer) ApplyUpdate(ctx context.Context, update *cmpb.FlightUpdate) (*cmpb.Empty, error) {
    s.applyUpdateLocal(update)
    s.sendGossip(update)
    return &cmpb.Empty{}, nil
}

func (s *DatanodeServer) GetFlightState(ctx context.Context, req *cmpb.GetFlightRequest) (*cmpb.FlightState, error) {

    s.mu.Lock()
    fs, ok := s.flights[req.FlightId]
    s.mu.Unlock()

    if !ok {
        return &cmpb.FlightState{
            FlightId:    req.FlightId,
            Status:      "",
            Gate:        "",
            VectorClock: map[string]int32{},
        }, nil
    }

    if isBehind(fs.VectorClock, req.ClientVersion) {
        return nil, status.Errorf(codes.FailedPrecondition, "stale_version")
    }

    resp := &cmpb.FlightState{
        FlightId:    fs.FlightId,
        Status:      fs.Status,
        Gate:        fs.Gate,
        VectorClock: copyVectorClock(fs.VectorClock),
    }

    return resp, nil
}
