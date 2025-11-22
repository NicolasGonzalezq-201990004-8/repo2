package main

import (
	"encoding/csv"
	"io"
	"log"
	"math/rand"
	"os"
	"time"

	cdpb "lab_3/proto/coordinator/cdpb"

	"google.golang.org/grpc"
	"google.golang.org/grpc/credentials/insecure"
)

const csvFilename = "flight_updates.csv"

func loadFlightCatalog() []string {
	log.Println("[Cliente RYW] Cargando catálogo de vuelos desde CSV...")

	file, err := os.Open(csvFilename)
	if err != nil {
		log.Fatalf("Error al abrir el archivo CSV %s: %v", csvFilename, err)
	}
	defer file.Close()

	reader := csv.NewReader(file)

	if _, err := reader.Read(); err != nil {
		log.Fatal(err)
	}

	uniqueFlights := make(map[string]bool)

	for {
		row, err := reader.Read()
		if err == io.EOF {
			break
		}
		if err != nil {
			continue
		}

		if len(row) > 1 {
			uniqueFlights[row[1]] = true
		}
	}

	flightList := make([]string, 0, len(uniqueFlights))
	for id := range uniqueFlights {
		flightList = append(flightList, id)
	}

	log.Printf("[Cliente RYW] Catálogo cargado. %d vuelos disponibles para reservar.", len(flightList))
	return flightList
}

func main() {
	coordAddr := os.Getenv("COORDINATOR_ADDR")
	if coordAddr == "" {
		log.Fatal("COORDINATOR_ADDR no configurada")
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		log.Fatal("CLIENT_ID NO CONFIGURADA")
		clientID = "Pasajero-Genérico"
	}

	availableFlights := loadFlightCatalog()
	if len(availableFlights) == 0 {
		log.Fatal("No se encontraron vuelos en el CSV. Abortando...")
	}

	rand.Seed(time.Now().UnixNano())

	log.Printf("Iniciando Cliente RYW [%s]. Conectando a %s...", clientID, coordAddr)

	conn, err := grpc.Dial(coordAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
	if err != nil {
		log.Fatalf("No se pudo conectar al Coordinador: %v", err)
	}
	defer conn.Close()

	client := &RYWClient{
		coordClient: cdpb.NewCoordinatorServiceClient(conn),
		clientID:    clientID,
	}

	time.Sleep(5 * time.Second)

	for {
		randomIndex := rand.Intn(len(availableFlights))
		targetFlight := availableFlights[randomIndex]

		client.RunPassengerScenario(targetFlight)

		wait := time.Duration(5+rand.Intn(6)) * time.Second
		log.Println("Esperando antes de la siguiente simulación...")
		time.Sleep(wait)
	}
}