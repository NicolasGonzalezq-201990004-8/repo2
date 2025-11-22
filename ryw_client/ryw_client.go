package main

import (
	"context"
	"fmt"
	"log"
	"time"

	cmpb "lab_3/proto/common/cmpb"
	cdpb "lab_3/proto/coordinator/cdpb"
)

type RYWClient struct {
	coordClient cdpb.CoordinatorServiceClient
	clientID    string
}

// RunPassengerScenario ejecuta el flujo completo de un pasajero
func (c *RYWClient) RunPassengerScenario(flightID string) {
	ctx := context.Background()

	log.Printf("[%s] Iniciando flujo para Vuelo %s...", c.clientID, flightID)

	seatsReq := &cmpb.SeatsRequest{FlightId: flightID}
	seatsResp, err := c.coordClient.GetSeats(ctx, seatsReq)
	if err != nil {
		log.Printf("[%s] Error consultando asientos: %v", c.clientID, err)
		return
	}
	log.Printf("[%s] Asientos ocupados vistos: %v", c.clientID, len(seatsResp.SeatsMap))

	targetSeat := fmt.Sprintf("%sA", c.clientID[len(c.clientID)-1:])

	log.Printf("[%s] Intentando Check-In en asiento %s...", c.clientID, targetSeat)

	checkInReq := &cmpb.CheckInRequest{
		FlightId:    flightID,
		Seat:        targetSeat,
		PassengerId: c.clientID,
	}

	start := time.Now()
	resp, err := c.coordClient.CheckIn(ctx, checkInReq)
	if err != nil {
		log.Printf("[%s] Falló el Check-In: %v", c.clientID, err)
		return
	}
	success := resp.Success
	if success {
		log.Printf("[%s] Check-In confirmado (Latencia: %v).", c.clientID, time.Since(start))
	} else {
		log.Printf("[%s] Check-In rechazado...", c.clientID)
	}

	log.Printf("[%s] Solicitando Tarjeta de Embarque (Verificación RYW)...", c.clientID)

	bpReq := &cmpb.BoardingPassRequest{
		FlightId:    flightID,
		PassengerId: c.clientID,
	}

	bpResp, err := c.coordClient.GetBoardingPass(ctx, bpReq)
	if err != nil {
		log.Printf("[%s] Error obteniendo Boarding Pass: %v", c.clientID, err)
		return
	}

	// VALIDACIÓN DE CONSISTENCIA
	if bpResp.Seat == targetSeat {
		log.Printf("[%s] ÉXITO RYW: La tarjeta muestra el asiento correcto (%s).", c.clientID, bpResp.Seat)
	} else {
		log.Printf("[%s] FALLO RYW: Inconsistencia detectada. Esperaba %s, recibió %s.", c.clientID, targetSeat, bpResp.Seat)
	}
}
