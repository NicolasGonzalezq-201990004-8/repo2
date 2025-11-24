package main

import (
	"context"
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
		coordAddr = "coordinator:50070"
		log.Printf("[WARN] COORDINATOR_ADDR no configurada, usando default local: %s", coordAddr)
	}

	clientID := os.Getenv("CLIENT_ID")
	if clientID == "" {
		clientID = "Pasajero-Genérico"
		log.Printf("[WARN] CLIENT_ID no configurado, usando identificador por defecto: %s", clientID)
	}

	availableFlights := loadFlightCatalog()
	if len(availableFlights) == 0 {
		log.Fatal("No se encontraron vuelos en el CSV. Abortando...")
	}

	rand.Seed(time.Now().UnixNano())

	log.Printf("Iniciando Cliente RYW [%s]. Conectando a %s...", clientID, coordAddr)

	conn := dialCoordinatorWithRetry(coordAddr)
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

func dialCoordinatorWithRetry(coordAddr string) *grpc.ClientConn {
	for attempt := 1; ; attempt++ {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)

		conn, err := grpc.DialContext(ctx, coordAddr, grpc.WithTransportCredentials(insecure.NewCredentials()), grpc.WithBlock())
		cancel()

		if err == nil {
			log.Printf("[Cliente RYW] Conectado al Coordinador en %s", coordAddr)
			return conn
		}

		log.Printf("[Cliente RYW] Reintento %d conectando al Coordinador: %v", attempt, err)
		time.Sleep(2 * time.Second)
	}
}
