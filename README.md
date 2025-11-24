# Aerodist - Laboratorio 3 de Sistemas Distribuidos (UTFSM)

**AeroDist** es un sistema distribuido que simula operaciones críticas de un aeropuerto moderno.


---

## Descripción de Carpetas y Archivos

### `proto/`
Contiene los archivos **.proto** que definen las interfaces gRPC y los mensajes entre servicios.

| Archivo | Propósito |
|----------|------------|
| **common.proto** | Estructuras base: estado de vuelo, reloj vectorial, mensajes comunes. |
| **datanode.proto** | Define el servicio `DatanodeService`: escritura, lectura y gossip entre nodos. |
| **broker.proto** | Define el servicio `BrokerService`: enruta solicitudes y maneja el consenso. |
| **coordinator.proto** | Servicio `CoordinatorService`: maneja sesiones para garantizar *Read Your Writes*. |
| **consensus.proto** | Servicio `ConsensusService`: implementa el algoritmo de consenso (Raft simplificado). |

Todos los servicios usan `import "common.proto"` para compartir estructuras.

---

###  `broker/` — *Sistema Central de Operaciones*

Orquesta las solicitudes entre los distintos subsistemas.  
Distribuye las actualizaciones, balanceador de carga y comunicación con los nodos de consenso.

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Inicializa el servidor gRPC. |
| `broker.go` | Implementa el servicio `BrokerService`. |
| `handlers.go` | Funciones auxiliares (broadcast, selección de nodo, logs, etc). |
| `Dockerfile` | Imagen del broker. |

**Funciones principales a implementar:**
- `LoadBalance`: Balancear la carga de las consultas de lectura recibidas de los clientes MR.
- `BroadcastUpdate`: Recibir actualizaciones de estado de vuelos (provenientes de las aerolíneas) y distribuirlas de forma asíncrona a los Datanodes.
- `RequestRunwayAssignment`: Simula solicitud de asignación de pista del nodo de consenso y envía la solicitud a los nodos de consenso.


**Funciones gRPC:**
- `QueryFlight`: Consulta estado de vuelo de cliente MR
- `GetDatanode`: Solicitud del coordinador para registrar afinidad.
- `GetSeats`: Solicitud de asientos disponibles.
- `CheckIn`: Solicitud de Check-in del coordinador.
- `GetBoardingPass`: Solicitud de Tarjeta de Embarque del coordinador

---

### `coordinator/` — *Gateway de Check-in*

Asegura que cada cliente vea sus propias escrituras inmediatamente.   
Comunica y coordina Clientes RYW con Coordinador y Broker.

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Inicia el servidor gRPC del Coordinador. |
| `coordinator.go` | Implementa el servicio `CoordinatorService` y el resto de funciones. |
| `session_map.go` | Administra el mapa de sesiones (TTL, afinidad por cliente). |
| `Dockerfile` | Imagen del Coordinador. |

**Funciones principales a implementar:**
 - `CheckAffinity`: Verifica si existe una sesión activa en su mapeo.
 - `RegisterAffinity`: Registra afinidad de cliente en su mapeo.

**Funciones gRPC:**
- `GetSeats`: Solicita asientos disponibles al broker.
- `CheckIn`: Solicita inicar el Check-in al broker.
- `GetBoardingPass`: Solicita Tarjeta de Embarque al broker.

---

### `datanode/` — *Servidores de Información de Vuelos*

Almacena y replicar el estado de los vuelos usando consistencia eventual.  

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Punto de entrada del nodo. |
| `datanode.go` | Implementación del servicio `DatanodeService`. |
| `vector_clock.go` | Estructura y operaciones del reloj vectorial (`Merge`, `Compare`, etc). |
| `gossip.go` | Comunicación periódica con otros nodos para propagación de actualizaciones. |
| `Dockerfile` | Imagen del Datanode (general para poder levantar 3 instancias). |

**Funciones principales a implementar**:
- `Merge`: Fusiona el reloj vectorial del evento con su reloj local.
- `Compare`: Al recibir una actualización, compara con la versión almacenada y resuelve apropiadamente.

**Funciones gRPC**:
- `ApplyUpdate`: Actualiza el estado del vuelo y fusiona el reloj vectorial del evento con su reloj local antes de aplicar el cambio.
- `GetFlightState`: Entrega el estado del vuelo.
- `Gossip`: iniciar comunicación con otro Datanode (elegido al azar) para intercambiar actualizaciones que el otro, propagando la información hasta alcanzar la convergencia.
- `GetSeats`
- `GetBoardingPass`
- `CheckIn`

---

### `consensus/` — *Nodos de Control de Tráfico Aéreo*

Garantiza consistencia fuerte en decisiones críticas (asignación de pistas).    
Elige líder, replicación de logs, quorum y commit.

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Inicializa el nodo de consenso. |
| `consensus.go` | Implementa el servicio `ConsensusService`. |
| `raft.go` | Lógica central de Raft: líder, votaciones, heartbeats. |
| `log.go` | Manejo del log replicado (historial de decisiones). |
| `Dockerfile` | Imagen de los nodos de consenso (general para poder levantar 3 instancias). |

**Funciones principales a implementar:**
- `UpdateNode`: Lider actualiza al nodo desactualizado.

**Funciones gRPC**:

**Públicas**:
- `RequestRunwayAssignment`: Solicitud del broker para asignar una pista.
**Internas**:
- `AppendEntry`: Replicar entradas del log y actuar como "heartbeat" para mantener el liderazgo. La llama el nodo lider. Si algún nodo rechaza esta llamada e intenta sincronizar al nodo.
- `RequestVote`: Iniciar una elección de líder. La llama un nodo candidato.


---

###  `ryw_client/` — *Simuladores de Pasajeros Read Your Writes (Pasajeros en Check-in)*

Probar los modelos de consistencia.  
Realizan operaciones de escritura y validan Read Your Writes.  

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Inicializa el cliente RYW e inicia conexiones gRPC |
| `ryw_client.go` | Implementación lógica de interacción del cliente RYW con sus funciones respectivas.|
| `Dockerfile` | Imagen para ejecutar clientes RYW (general para poder levantar 3 instancias). |

**Funciones principales a implementar:**
- `ValidateRYW`: Validación de Consistencia.

**Funciones gRPC**:
- `GetSeats`: Solicitud de estado inicial, consultando los asientos disponibles.
- `CheckIn`: Solicitud de CheckIn al Coordinador (envío operación de escritura).
- `GetBoardingPass`: Solicitud de Tarjeta de Embarque
---

###  `mr_client/` — *Simuladores de Pasajeros Monotonic Reads (Pasajeros Observadores)*

Probar los modelos de consistencia.  
Simulan a los usuarios que consumen información pública de manera pasiva

| Archivo | Descripción |
|----------|-------------|
| `main.go` | Inicializa el cliente MR e inicia conexiones gRPC|
| `mr_client.go` | Estructura MRClient con `knowkVersions` `map[string]VectorClock`. Implementa funciones, etc.|
| `Dockerfile` |Imagen para ejecutar clientes MR (general para poder levantar 2 instancias). |

**Funciones principales a implementar:**
- `knowkVersions`: `map[string]VectorClock` con la última versión vista del dato.
- `QueryFlight`: Consulta el estado algún vuelo. $\rightarrow$ solicitud gRPC al Broker.



---

### 🐳 `docker-compose.yml`
Define la red completa del laboratorio en un único host.
Todos los servicios se comunican usando los nombres de los contenedores (por ejemplo, `broker:50050` o `datanode2:50051`), de modo que puedes levantar el clúster completo con un solo comando.

Puertos expuestos en el host para depuración rápida:

| Servicio      | Puerto host -> contenedor |
| ------------- | -------------------------- |
| Broker        | 50050 -> 50050            |
| Datanode 1    | 50051 -> 50051            |
| Datanode 2    | 50052 -> 50051            |
| Datanode 3    | 50053 -> 50051            |
| Consensus 1   | 50060 -> 50060            |
| Consensus 2   | 50061 -> 50060            |
| Consensus 3   | 50062 -> 50060            |
| Coordinator   | 50070 -> 50070            |

Los clientes (`mr_client*` y `ryw_client*`) se conectan internamente usando los nombres de los servicios, sin exponer puertos adicionales.

---

### 🧰 `Makefile`
Atajos para el nuevo modo local:

```bash
make docker-up     # Construye y levanta todo el clúster en esta máquina
make docker-down   # Detiene y elimina los contenedores
make docker-logs   # Sigue los logs de todos los servicios
make clean         # Limpia artefactos generados por los .proto
```

> Ya no se requieren múltiples perfiles o IPs de distintas VMs. Todo se resuelve en la red de Docker local usando los nombres de los servicios.

Los contenedores ya exponen los puertos gRPC directamente en la IP de cada host para que las instancias en otras VMs puedan conectarse (p. ej. broker en `10.35.168.15:50050`, datanodes en `:50051` y nodos de consenso en `:50060`).

---
# Distribución del Trabajo
Cada uno elegir un rol que esté disponible
| Integrante               | Responsabilidad                     
| ------------------------ | ------------------------------------------------------------ |
| José Astudillo | Nodos de Consenso + Broker Central |
| Integrante 2         |    Datanodes + Clientes MR    | 
| Integrante 3          |  Coordinador + Clientes RYW |


