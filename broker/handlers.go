package main

import (
	"context"
	"encoding/csv"
	"fmt"
	"io"
	cmpb "lab_3/proto/common/cmpb"
	dpb "lab_3/proto/datanode/dpb"
	"log"
	"os"
	"strconv"
	"time"
)

const (
	csvFilename = "flight_updates.csv"
)

// selectNextDatanode implementa la lógica de balanceo de carga (Round Robin)
// Es un método de BrokerServer para acceder al estado (clientes e índice)
func (s *BrokerServer) selectNextDatanode() (dpb.DatanodeServiceClient, string) {
	// Bloqueamos el mutex para acceder/modificar el índice de forma segura
	s.mu.Lock()
	// Diferimos el desbloqueo
	defer s.mu.Unlock()

	if len(s.datanodeClients) == 0 {
		log.Println("Error: No hay Datanodes disponibles.")
		return nil, ""
	}

	idx := s.datanodeIndex

	// Lógica de Round Robin
	s.datanodeIndex = (s.datanodeIndex + 1) % len(s.datanodeClients)

	client := s.datanodeClients[idx]
	addr := s.datanodeAddresses[idx]
	return client, addr
}

func (s *BrokerServer) StartEventSimulation(ctx context.Context) {
	log.Printf("[Eventos] Iniciando lectura de eventos.")

	file, err := os.Open(csvFilename)
	if err != nil {
		log.Fatalf("Error al abrir el archivo CSV %s: %v", csvFilename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	reader.Read()

	var lastSimTime int64 = 0

	for {
		row, err := reader.Read()
		if err == io.EOF {
			log.Println("[Eventos] Fin del archivo CSV. Reiniciando eventos.")
			file.Seek(0, io.SeekStart)
			reader = csv.NewReader(file)
			reader.Read()
			lastSimTime = 0
			continue
		}

		simTime, err := strconv.ParseInt(row[0], 10, 64)
		if err != nil {
			log.Printf("[Eventos] Error al parsear sim_time_sec: %v. Saltando fila.", err)
			continue
		}
		pauseDuration := simTime - lastSimTime
		if pauseDuration > 0 {
			log.Printf("[Eventos] Esperando evento...")
			time.Sleep(time.Duration(pauseDuration) * time.Second)
		}
		lastSimTime = simTime

		flightID := row[1]
		updateType := row[2]
		updateValue := row[3]

		// --- Lógica Reloj Vectorial ---
		s.mu.Lock()
		s.flightCounters[flightID]++
		currentCount := s.flightCounters[flightID]
		s.mu.Unlock()

		vcMap := map[string]int32{
			s.brokerID: currentCount,
		}
		// ------------------------------

		updateMsg := &cmpb.FlightUpdate{
			FlightId:    flightID,
			Status:      "~NULL~",
			Gate:        "~NULL~",
			VectorClock: vcMap,
		}

		switch updateType {
		case "estado":
			updateMsg.Status = updateValue
		case "puerta":
			updateMsg.Gate = updateValue
		default:
			log.Printf("[Eventos] Tipo de actualización desconocido: %s", updateType)
		}

		s.BroadcastUpdate(ctx, updateMsg) //función que hace broadcast a los datanodes

		//		if updateValue == "Llegó" || updateValue == "Embarcando" {
		//			log.Printf("[Eventos] Vuelo %s requiere PISTA. Iniciando consenso...", updateMsg.FlightId)
		//			go s.RequestRunwayAssignment(context.Background(), updateMsg.FlightId)
		//		}

	}

}

func (s *BrokerServer) loadFlightCatalog() {
	log.Println("[Broker] Cargando catálogo maestro de vuelos...")

	file, err := os.Open(csvFilename)
	if err != nil {
		log.Fatalf("Error al abrir el archivo CSV %s: %v", csvFilename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)
	reader.Read()

	uniqueFlights := make(map[string]bool)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			log.Printf("[Broker] Error leyendo fila: %v", err)
			continue
		}

		flightID := row[1]
		uniqueFlights[flightID] = true
	}
	s.mu.Lock()
	for flightID := range uniqueFlights {
		s.flightCounters[flightID] = 0
	}
	s.mu.Unlock()

	log.Printf("[Broker] Catálogo cargado. %d vuelos únicos inicializados.", len(uniqueFlights))
}

func (s *BrokerServer) ReportWriter() {
	file, err := os.OpenFile(ReportFilename, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
	if err != nil {
		log.Fatalf("No se pudo abrir el archivo de reporte: %v", err)
	}
	defer file.Close()

	log.Printf("[Reporte] Iniciando reporte en %s...", ReportFilename)

	for entry := range s.reportChan {
		reportEntry := fmt.Sprintf("[%s] %s\n", time.Now().Format("15:05:05.000"), entry)

		if _, err := file.WriteString(reportEntry); err != nil {
			log.Printf("[Reporte] Error crítico al escribir en el archivo: %v", err)
		}
	}
}
