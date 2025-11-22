package main

import (
	"encoding/csv"
	"fmt"
	"log"
	"math/rand"
	"os"
	"time"
)

// -----------------------------
// Cargar lista de vuelos del CSV
// -----------------------------
func loadFlights() ([]string, error) {
	file, err := os.Open("/app/flight_updates.csv")
	if err != nil {
		return nil, err
	}
	defer file.Close()

	r := csv.NewReader(file)
	rows, err := r.ReadAll()
	if err != nil {
		return nil, err
	}

	var ids []string
	for i, row := range rows {
		if i == 0 {
			continue
		}
		ids = append(ids, row[1]) // columna flight_id
	}
	return ids, nil
}

// -----------------------------
// WORKER: Cliente MR independiente
// -----------------------------
func mrWorker(workerID int, brokerAddr string, flights []string) {

	client, err := NewMRClient(brokerAddr)
	if err != nil {
		log.Fatalf("[Worker %d] Error creando cliente MR: %v", workerID, err)
	}

	log.Printf("[Worker %d] Iniciado correctamente", workerID)

	for {
		// 1. Elegir vuelo al azar
		flightID := flights[rand.Intn(len(flights))]

		// 2. Consultar
		resp, err := client.QueryFlightMR(flightID)
		if err != nil {
			log.Printf("[Worker %d] Error consultando vuelo %s: %v", workerID, flightID, err)
		} else {
			fmt.Printf("\n===== WORKER %d — MONOTONIC READ =====\n", workerID)
			fmt.Printf("Vuelo:   %s\n", resp.FlightId)
			fmt.Printf("Estado:  %s\n", resp.Status)
			fmt.Printf("Gate:    %s\n", resp.Gate)
			fmt.Printf("VC:      %v\n", resp.VectorClock)
			fmt.Println("=======================================")
		}

		time.Sleep(3 * time.Second)
	}
}

func main() {

	brokerAddr := os.Getenv("BROKER_ADDR")
	if brokerAddr == "" {
		log.Fatal("BROKER_ADDR no definido")
	}

	rand.Seed(time.Now().UnixNano())

	// Cargar lista de vuelos
	flights, err := loadFlights()
	if err != nil || len(flights) == 0 {
		log.Fatalf("Error cargando vuelos del CSV: %v", err)
	}

	log.Printf("Vuelos disponibles: %v", flights)

	// Lanzar dos clientes en paralelo
	go mrWorker(1, brokerAddr, flights)

	select {}
}
