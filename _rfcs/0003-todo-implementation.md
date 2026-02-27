# RFC 0003: User Stories e Tasks de Implementação
**Data:** 21 de Fevereiro de 2026
**Versão:** 1.2.0

---

## Stack Tecnológica

- **Linguagem:** Go 1.21+
- **ORM:** GORM
- **Banco de Dados:** PostgreSQL 15+ (com PostGIS), Redis 7+
- **Padrão de Dados:** Database-per-Service (cada microsserviço possui sua própria instância de PostgreSQL)
- **Message Broker:** RabbitMQ 3.12+
- **Orquestração Local:** Minikube + Tilt
- **API Gateway:** Kong
- **Observabilidade:** Prometheus, Grafana

---

## Índice

1. [Infraestrutura e Setup Inicial](#1-infraestrutura-e-setup-inicial)
2. [User Service](#2-user-service)
3. [Location Service](#3-location-service)
4. [Ride Service](#4-ride-service)
5. [Matching Service](#5-matching-service)
6. [Notification Service](#6-notification-service)
7. [API Gateway](#7-api-gateway)
8. [Integrações e Testes E2E](#8-integrações-e-testes-e2e)
9. [Observabilidade e Monitoramento](#9-observabilidade-e-monitoramento)

---

## 1. Infraestrutura e Setup Inicial

### US-001: Setup de Ambiente de Desenvolvimento com Minikube e Tilt
**Como** desenvolvedor
**Quero** ter um ambiente de desenvolvimento local Kubernetes completo
**Para que** eu possa desenvolver e testar os microsserviços em ambiente próximo a produção

#### Critérios de Aceite:
- Minikube rodando localmente com todos os serviços
- Tilt configurado para hot-reload de serviços Go
- Scripts de inicialização automatizados
- Seed data para desenvolvimento
- Documentação de setup atualizada

#### Tasks:
- [x] **Task 1.1.1:** Instalar e configurar Minikube
  - Instalar Minikube: `brew install minikube` (macOS) ou equivalente
  - Iniciar cluster: `minikube start --memory=8192 --cpus=4`
  - Configurar Docker local para usar registry do Minikube: `eval $(minikube docker-env)`

- [x] **Task 1.1.2:** Criar estrutura de diretórios Kubernetes
  ```
  k8s/
  ├── development/
  │   ├── base/
  │   │   ├── postgres/
  │   │   │   ├── ride-db/
  │   │   │   │   ├── deployment.yaml
  │   │   │   ├── user-db/
  │   │   │   │   ├── deployment.yaml
  │   │   ├── redis/
  │   │   │   ├── deployment.yaml
  │   │   ├── rabbitmq/
  │   │   │   ├── deployment.yaml
  │   │   └── appconfig.yaml
  │   │   └── secrets.yaml
  └── services/
      ├── user-service/
      ├── location-service/
      ├── ride-service/
      ├── matching-service/
      └── notification-service/
  ```

- [x] **Task 1.1.3:** Criar manifests Kubernetes para PostgreSQL (múltiplas instâncias para Database-per-Service)
  - Criar deployment para **user-service-db**
  - Criar deployment para **ride-service-db**
  - Cada serviço possui sua própria instância isolada

- [x] **Task 1.1.4:** Criar manifests Kubernetes para Redis
  
- [x] **Task 1.1.5:** Criar manifests Kubernetes para RabbitMQ

- [x] **Task 1.1.6:** Criar Tiltfile principal

- [x] **Task 1.1.7:** Criar Makefile com comandos Tilt

- [x] **Task 1.1.8:** Documentar requisitos e setup no README.md

---

### US-002: Modelagem do Banco de Dados PostgreSQL (Database-per-Service)
**Como** desenvolvedor
**Quero** ter o schema de banco de dados implementado em cada microsserviço
**Para que** cada serviço possa gerenciar seus dados de forma autônoma seguindo o padrão Database-per-Service

#### Critérios de Aceite:
- Cada microsserviço possui sua própria instância de PostgreSQL
- **User Service:** Tabelas `users` no banco `user-service-db`
- **Ride Service:** Tabelas `rides`, `ride_updates`, `fares` no banco `ride-service-db`
- **Location Service:** Utiliza Redis como storage principal (sem PostgreSQL)

- Nenhum serviço acessa diretamente o banco de outro serviço
- Dados compartilhados são trocados via eventos (RabbitMQ) ou APIs REST/gRPC
 - Dados compartilhados são trocados via eventos (RabbitMQ) ou APIs REST/gRPC
 - Não usar relacionamentos GORM que referenciem modelos de outros serviços; cada serviço deve manter apenas os identificadores (UUIDs) de recursos externos e resolver dados via API/eventos
 - Índices otimizados para queries principais
- Constraints e foreign keys configurados dentro de cada banco
- Migrations versionadas por serviço

#### Tasks:
- [x] **Task 1.2.1:** Configurar GORM Auto Migration
  - Usar GORM Auto Migrate para criar/atualizar tabelas automaticamente
  - Estruturar diretório: `internal/models/`
  - Criar script de inicialização de database
  - Observação: não incluir models de outros serviços nas migrations (ex.: não auto-migrar `User` no `ride-service`). Manter apenas tabelas pertencentes ao serviço.

- [x] **Task 1.2.2:** Criar modelo GORM para tabela `users`


- [ ] **Task 1.2.3:** Criar modelo GORM para tabela `rides`
  ```go
  // internal/models/ride.go
  package models

  import (
    "time"
    "github.com/google/uuid"
    "gorm.io/gorm"
  )

  type RideStatus string

  const (
    RideStatusRequested  RideStatus = "REQUESTED"
    RideStatusMatching   RideStatus = "MATCHING"
    RideStatusAccepted   RideStatus = "ACCEPTED"
    RideStatusInProgress RideStatus = "IN_PROGRESS"
    RideStatusCompleted  RideStatus = "COMPLETED"
    RideStatusCancelled  RideStatus = "CANCELLED"
  )

  type Ride struct {
    ID                   uuid.UUID      `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    RiderID              uuid.UUID      `gorm:"type:uuid;not null;index"`
    DriverID             *uuid.UUID     `gorm:"type:uuid;index"`
    FareID               *uuid.UUID     `gorm:"type:uuid"`
    Status               RideStatus     `gorm:"type:varchar(20);not null;default:'REQUESTED';index"`
    PickupLocation       string         `gorm:"type:geometry(Point,4326);not null"` // PostGIS
    DestinationLocation  string         `gorm:"type:geometry(Point,4326);not null"` // PostGIS
    EstimatedFare        *float64       `gorm:"type:decimal(10,2)"`
    FinalFare            *float64       `gorm:"type:decimal(10,2)"`
    RequestedAt          time.Time      `gorm:"autoCreateTime;index"`
    MatchedAt            *time.Time
    StartedAt            *time.Time
    CompletedAt          *time.Time
    CancelledAt          *time.Time
    CancellationReason   string         `gorm:"type:text"`
    CreatedAt            time.Time      `gorm:"autoCreateTime"`
    UpdatedAt            time.Time      `gorm:"autoUpdateTime"`

    // Observação: não referenciar structs/relationships de outros serviços aqui.
    // Manter apenas os IDs (RiderID, DriverID) e buscar dados via User Service quando necessário.
  }

  func (Ride) TableName() string {
    return "rides"
  }

  // GORM Hook para criar índice GIST no PostGIS
  func (Ride) AfterAutoMigrate(tx *gorm.DB) error {
    return tx.Exec("CREATE INDEX IF NOT EXISTS idx_rides_pickup_location ON rides USING GIST(pickup_location)").Error
  }
  ```

- [ ] **Task 1.2.4:** Criar modelo GORM para tabela `ride_updates` (immutable ledger)
  ```go
  // internal/models/ride_update.go
  package models

  import (
    "time"
    "github.com/google/uuid"
    "gorm.io/datatypes"
  )

  type RideUpdate struct {
    ID        uint64         `gorm:"primaryKey;autoIncrement"`
    RideID    uuid.UUID      `gorm:"type:uuid;not null;index:idx_ride_updates_ride_id"`
    OldStatus *RideStatus    `gorm:"type:varchar(20)"`
    NewStatus RideStatus     `gorm:"type:varchar(20);not null"`
    ChangedAt time.Time      `gorm:"autoCreateTime;index:idx_ride_updates_ride_id"`
    ChangedBy *uuid.UUID     `gorm:"type:uuid"`
    Metadata  datatypes.JSON `gorm:"type:jsonb"`

    // Observação: não referenciar structs/relationships de outros serviços aqui.
    // Usar apenas os IDs (RideID, ChangedBy) — o relacionamento com `rides` é interno ao serviço
  }

  func (RideUpdate) TableName() string {
    return "ride_updates"
  }

  // GORM Hook para criar índice GIN no JSONB
  func (RideUpdate) AfterAutoMigrate(tx *gorm.DB) error {
    return tx.Exec("CREATE INDEX IF NOT EXISTS idx_ride_updates_metadata ON ride_updates USING GIN(metadata)").Error
  }
  ```

- [ ] **Task 1.2.5:** Criar modelo GORM para tabela `fares`
  ```go
  // internal/models/fare.go
  package models

  import (
    "time"
    "github.com/google/uuid"
  )

  type Fare struct {
    ID                   uuid.UUID `gorm:"type:uuid;primary_key;default:gen_random_uuid()"`
    PickupLocation       string    `gorm:"type:geometry(Point,4326);not null"`
    DestinationLocation  string    `gorm:"type:geometry(Point,4326);not null"`
    DistanceMeters       int       `gorm:"not null"`
    DurationSeconds      int       `gorm:"not null"`
    BaseFare             float64   `gorm:"type:decimal(10,2);not null"`
    DistanceFare         float64   `gorm:"type:decimal(10,2);not null"`
    TimeFare             float64   `gorm:"type:decimal(10,2);not null"`
    SurgeMultiplier      float64   `gorm:"type:decimal(3,2);default:1.0"`
    EstimatedTotal       float64   `gorm:"type:decimal(10,2);not null"`
    ValidUntil           time.Time `gorm:"not null;index"`
    CreatedAt            time.Time `gorm:"autoCreateTime;index"`
  }

  func (Fare) TableName() string {
    return "fares"
  }
  ```

- [ ] **Task 1.2.6:** Configurar GORM Hooks para auditoria automática
  ```go
  // internal/models/hooks.go
  package models

  import "gorm.io/gorm"

  // BeforeUpdate hook para atualizar updated_at automaticamente
  // (GORM já faz isso com autoUpdateTime, mas podemos customizar)

  // Hook para registrar mudanças de status da ride
  func (r *Ride) AfterUpdate(tx *gorm.DB) error {
    // Verificar se o status mudou
    var oldRide Ride
    if err := tx.Model(&Ride{}).Where("id = ?", r.ID).First(&oldRide).Error; err != nil {
      return err
    }

    if oldRide.Status != r.Status {
      rideUpdate := RideUpdate{
        RideID:    r.ID,
        OldStatus: &oldRide.Status,
        NewStatus: r.Status,
      }
      return tx.Create(&rideUpdate).Error
    }
    return nil
  }
  ```

- [ ] **Task 1.2.7:** Criar script de inicialização do banco com GORM

---

### US-003: Setup de Redis e Estratégia de Chaves
**Como** desenvolvedor
**Quero** ter a estrutura de chaves do Redis documentada e configurada
**Para que** o Location Service possa gerenciar geo-índices e locks de forma consistente

#### Critérios de Aceite:
- Padrões de nomenclatura de chaves documentados
- Configuração de persistência (AOF/RDB) definida
- Scripts de setup e limpeza disponíveis

#### Tasks:
- [ ] **Task 1.3.1:** Documentar padrões de chaves Redis em `docs/redis-keys.md`
  ```
  Geo-Index:
  - drivers:geo:available -> GEOSPATIAL SET (lon, lat, driver_id)

  Locks:
  - driver:{driver_id}:lock -> STRING (ride_id) [TTL: 15s]

  Driver Status:
  - driver:{driver_id}:status -> HASH
    - status: "available" | "busy" | "offline"
    - last_ping: timestamp
    - current_ride_id: UUID | null

  Rate Limiting (Location Service):
  - ratelimit:location:{driver_id}:{minute} -> COUNTER [TTL: 60s]
  ```

- [ ] **Task 1.3.2:** Criar configuração Redis em `redis/redis.conf`
  ```conf
  maxmemory 2gb
  maxmemory-policy allkeys-lru
  appendonly yes
  appendfsync everysec
  save 900 1
  save 300 10
  ```

- [ ] **Task 1.3.4:** Implementar helper para geração de chaves em Go
  ```go
  package rediskeys

  func DriverLock(driverID string) string
  func DriverStatus(driverID string) string
  func GeoIndexKey() string
  ```

---

## 2. User Service

### US-004: CRUD de Usuários (Riders e Drivers) com GORM
**Como** sistema
**Quero** gerenciar cadastro e perfis de passageiros e motoristas
**Para que** outros serviços possam referenciar usuários válidos

#### Critérios de Aceite:
- Endpoints REST funcionais para CRUD
- Validação de dados de entrada
- Soft delete implementado via GORM
- Testes unitários com cobertura > 80%

#### Tasks:
- [x] **Task 2.1.1:** Setup do projeto User Service
  - Criar estrutura de diretórios: `services/user-service/`
  - Configurar dependências:
    - `github.com/gin-gonic/gin` (HTTP framework)
    - `gorm.io/gorm` (ORM)
    - `gorm.io/driver/postgres` (PostgreSQL driver)
    - `github.com/google/uuid` (UUID generation)

- [x] **Task 2.1.2:** Implementar models com GORM
- [x] **Task 2.1.3:** Implementar repository layer com GORM
- [x] **Task 2.1.4:** Implementar service layer
- [x] **Task 2.1.5:** Implementar controller layer com GRPC

---

### US-005: Health Check e Readiness do User Service
**Como** operador de infraestrutura  
**Quero** endpoints de health check  
**Para que** orquestradores (K8s) possam monitorar o serviço

#### Critérios de Aceite:
- `/health/live` retorna 200 quando o serviço está rodando
- `/health/ready` retorna 200 quando DB está acessível
- Timeout configurável para checks

#### Tasks:
- [ ] **Task 2.2.1:** Implementar endpoint `/health/live`
  ```go
  func (h *HealthHandler) Liveness(c *gin.Context) {
    c.JSON(200, gin.H{"status": "ok"})
  }
  ```
  
- [ ] **Task 2.2.2:** Implementar endpoint `/health/ready`
  ```go
  func (h *HealthHandler) Readiness(c *gin.Context) {
    ctx, cancel := context.WithTimeout(c.Request.Context(), 2*time.Second)
    defer cancel()
    
    if err := h.db.PingContext(ctx); err != nil {
      c.JSON(503, gin.H{"status": "not_ready", "error": err.Error()})
      return
    }
    c.JSON(200, gin.H{"status": "ready"})
  }
  ```
  
- [ ] **Task 2.2.3:** Configurar rotas de health em porta separada (8081)
  - Separar health checks do tráfego de aplicação
  - Evitar exposição pública de health endpoints

---

## 3. Location Service

### US-006: Ingestão de Pings de GPS via WebSocket
**Como** motorista  
**Quero** enviar minha localização em tempo real  
**Para que** o sistema saiba minha posição para matching

#### Critérios de Aceite:
- WebSocket endpoint aceita conexões autenticadas
- Pings são validados e processados com latência < 100ms (p99)
- Posições são indexadas no Redis via GEOADD
- Suporte a 10k conexões simultâneas por instância

#### Tasks:
- [ ] **Task 3.1.1:** Setup do projeto Location Service
  - Estrutura de diretórios: `services/location-service/`
  - Dependências Go:
    - `github.com/gorilla/websocket` (WebSocket)
    - `github.com/go-redis/redis/v9` (Redis client)
    - `github.com/gin-gonic/gin` (HTTP framework)
  
- [ ] **Task 3.1.2:** Implementar WebSocket handler
  ```go
  // internal/websocket/handler.go
  type LocationPing struct {
    DriverID  string    `json:"driver_id" validate:"required,uuid"`
    Latitude  float64   `json:"latitude" validate:"required,min=-90,max=90"`
    Longitude float64   `json:"longitude" validate:"required,min=-180,max=180"`
    Timestamp time.Time `json:"timestamp" validate:"required"`
    Heading   float64   `json:"heading" validate:"min=0,max=360"`
    SpeedKmh  float64   `json:"speed_kmh" validate:"min=0"`
  }
  
  func (h *WSHandler) HandleDriverConnection(c *gin.Context) {
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    // ... handle connection lifecycle
  }
  ```
  
- [ ] **Task 3.1.3:** Implementar processador de pings assíncrono
  - Canal Go com buffer: `pingQueue := make(chan LocationPing, 10000)`
  - Worker pool para processar pings em paralelo (10 workers)
  - Batching de GEOADD para otimizar throughput
  
- [ ] **Task 3.1.4:** Integrar com Redis GeoIndex
  ```go
  func (r *RedisRepository) UpdateDriverLocation(ctx context.Context, ping LocationPing) error {
    key := "drivers:geo:available"
    return r.client.GeoAdd(ctx, key, &redis.GeoLocation{
      Name:      ping.DriverID,
      Longitude: ping.Longitude,
      Latitude:  ping.Latitude,
    }).Err()
  }
  ```
  
- [ ] **Task 3.1.5:** Atualizar status do motorista (last_ping)
  ```go
  func (r *RedisRepository) UpdateDriverStatus(ctx context.Context, driverID string) error {
    key := fmt.Sprintf("driver:%s:status", driverID)
    return r.client.HSet(ctx, key, 
      "last_ping", time.Now().Unix(),
      "status", "available",
    ).Err()
  }
  ```
  
- [ ] **Task 3.1.6:** Implementar heartbeat e detecção de desconexão
  - Cliente envia ping a cada 3 segundos
  - Servidor marca motorista como offline após 15s sem ping
  - Remover motorista do geo-index quando ficar offline
  
- [ ] **Task 3.1.7:** Publicar eventos no RabbitMQ
  - Evento `location.driver.updated` a cada ping processado
  - Evento `location.driver.online` quando motorista conecta
  - Evento `location.driver.offline` quando desconecta

---

### US-007: API REST para Busca de Motoristas Próximos
**Como** Matching Service  
**Quero** buscar motoristas disponíveis por proximidade  
**Para que** eu possa enviar ofertas de corrida

#### Critérios de Aceite:
- Endpoint retorna motoristas ordenados por distância
- Filtros: raio (km), limite de resultados, status
- Latência p99 < 50ms
- Apenas motoristas "available" e sem lock

#### Tasks:
- [ ] **Task 3.2.1:** Implementar endpoint `GET /internal/drivers/nearby`
  ```go
  type NearbyRequest struct {
    Latitude  float64 `form:"lat" binding:"required"`
    Longitude float64 `form:"lng" binding:"required"`
    Radius    float64 `form:"radius" binding:"required,min=100,max=50000"` // metros
    Limit     int     `form:"limit" binding:"max=50" default:"10"`
  }
  
  type NearbyDriver struct {
    DriverID      string    `json:"driver_id"`
    DistanceMeters float64   `json:"distance_meters"`
    LastSeen      time.Time `json:"last_seen"`
    Status        string    `json:"status"`
  }
  ```
  
- [ ] **Task 3.2.2:** Implementar query Redis GEORADIUS
  ```go
  func (r *RedisRepository) FindNearbyDrivers(ctx context.Context, req NearbyRequest) ([]NearbyDriver, error) {
    results, err := r.client.GeoRadius(ctx, "drivers:geo:available", 
      req.Longitude, req.Latitude, &redis.GeoRadiusQuery{
        Radius:      req.Radius,
        Unit:        "m",
        WithDist:    true,
        Count:       req.Limit,
        Sort:        "ASC",
      },
    ).Result()
    // ... parse results
  }
  ```
  
- [ ] **Task 3.2.3:** Filtrar motoristas locked
  - Para cada resultado do GEORADIUS, verificar se existe chave `driver:{id}:lock`
  - Remover da lista se estiver locked
  - Otimização: usar pipeline Redis para verificar múltiplos locks em batch
  
- [ ] **Task 3.2.4:** Enriquecer resposta com metadados
  - Buscar `driver:{id}:status` para pegar last_ping
  - Validar que last_ping está dentro de 15 segundos (motorista ativo)
  - Retornar status atual do motorista

---

### US-008: Distributed Locking de Motoristas
**Como** Matching Service  
**Quero** adquirir lock exclusivo em um motorista  
**Para que** apenas uma corrida possa ser oferecida por vez

#### Critérios de Aceite:
- Lock é atômico (SET NX)
- TTL configurável (padrão 15s)
- Lock pode ser liberado manualmente ou expira automaticamente
- Endpoint retorna sucesso/falha claramente

#### Tasks:
- [ ] **Task 3.3.1:** Implementar endpoint `POST /internal/drivers/:id/lock`
  ```go
  type LockRequest struct {
    RideID     string `json:"ride_id" binding:"required,uuid"`
    TTLSeconds int    `json:"ttl_seconds" binding:"min=5,max=60" default:"15"`
  }
  
  type LockResponse struct {
    Locked    bool       `json:"locked"`
    ExpiresAt *time.Time `json:"expires_at,omitempty"`
    Reason    string     `json:"reason,omitempty"`
  }
  ```
  
- [ ] **Task 3.3.2:** Implementar lógica de aquisição de lock
  ```go
  func (r *RedisRepository) AcquireLock(ctx context.Context, driverID, rideID string, ttl time.Duration) (bool, error) {
    key := fmt.Sprintf("driver:%s:lock", driverID)
    result, err := r.client.SetNX(ctx, key, rideID, ttl).Result()
    return result, err
  }
  ```
  
- [ ] **Task 3.3.3:** Implementar endpoint `DELETE /internal/drivers/:id/lock`
  ```go
  func (r *RedisRepository) ReleaseLock(ctx context.Context, driverID, rideID string) error {
    key := fmt.Sprintf("driver:%s:lock", driverID)
    
    // Usar Lua script para garantir atomicidade (só deleta se o valor corresponder)
    script := `
      if redis.call("get", KEYS[1]) == ARGV[1] then
        return redis.call("del", KEYS[1])
      else
        return 0
      end
    `
    return r.client.Eval(ctx, script, []string{key}, rideID).Err()
  }
  ```
  
- [ ] **Task 3.3.4:** Implementar endpoint `GET /internal/drivers/:id/status`
  - Retornar se motorista está locked e para qual ride_id
  - Retornar tempo restante do lock (TTL)
  - Retornar status atual (available, busy, offline)
  
- [ ] **Task 3.3.5:** Adicionar métricas de lock
  - Counter: locks adquiridos com sucesso
  - Counter: tentativas de lock rejeitadas
  - Histogram: tempo que locks ficam ativos
  - Gauge: número de motoristas locked no momento

---

### US-009: Gestão de Status de Motoristas
**Como** sistema  
**Quero** gerenciar o ciclo de vida do status de motoristas  
**Para que** apenas motoristas elegíveis sejam considerados no matching

#### Critérios de Aceite:
- Motoristas podem ser marcados como available, busy, offline
- Motoristas offline são removidos do geo-index
- Motoristas busy não aparecem em buscas de proximidade
- Transições de estado são auditadas

#### Tasks:
- [ ] **Task 3.4.1:** Implementar endpoint `PATCH /internal/drivers/:id/status`
  ```go
  type UpdateStatusRequest struct {
    Status        string  `json:"status" binding:"required,oneof=available busy offline"`
    CurrentRideID *string `json:"current_ride_id,omitempty"`
  }
  ```
  
- [ ] **Task 3.4.2:** Implementar lógica de transição de estado
  ```go
  func (s *StatusService) UpdateStatus(ctx context.Context, driverID string, newStatus string) error {
    switch newStatus {
    case "available":
      // Verificar se não há lock ativo
      // Manter no geo-index
    case "busy":
      // Remover do geo-index temporariamente
      // Manter last_ping ativo
    case "offline":
      // Remover do geo-index
      // Limpar last_ping
      // Fechar conexão WebSocket se existir
    }
  }
  ```
  
- [ ] **Task 3.4.3:** Implementar job de limpeza de motoristas inativos
  - Cronjob a cada 30 segundos
  - Buscar motoristas com last_ping > 15 segundos
  - Marcar como offline automaticamente
  - Publicar evento `location.driver.offline`
  
- [ ] **Task 3.4.4:** Publicar eventos de mudança de status
  - `location.driver.online` quando available
  - `location.driver.offline` quando offline
  - Incluir timestamp e metadata no evento

---

## 4. Ride Service

### US-010: Estimativa de Tarifa
**Como** passageiro  
**Quero** saber o preço estimado da corrida  
**Para que** eu possa decidir se solicito ou não

#### Critérios de Aceite:
- Endpoint calcula tarifa baseado em distância e tempo estimado
- Tarifa inclui base fare + distance fare + time fare + surge
- Estimativa é válida por 5 minutos
- Tarifa é persistida para referência futura

#### Tasks:
- [ ] **Task 4.1.1:** Setup do projeto Ride Service
  - Estrutura: `services/ride-service/`
  - Dependências: gin, sqlx, amqp (RabbitMQ), redis
  
- [ ] **Task 4.1.2:** Implementar endpoint `POST /v1/fares`
  ```go
  type FareRequest struct {
    PickupLat     float64 `json:"pickup_lat" binding:"required"`
    PickupLng     float64 `json:"pickup_lng" binding:"required"`
    DestinationLat float64 `json:"destination_lat" binding:"required"`
    DestinationLng float64 `json:"destination_lng" binding:"required"`
  }
  
  type FareResponse struct {
    FareID          string    `json:"fare_id"`
    DistanceMeters  int       `json:"distance_meters"`
    DurationSeconds int       `json:"duration_seconds"`
    BaseFare        float64   `json:"base_fare"`
    DistanceFare    float64   `json:"distance_fare"`
    TimeFare        float64   `json:"time_fare"`
    SurgeMultiplier float64   `json:"surge_multiplier"`
    EstimatedTotal  float64   `json:"estimated_total"`
    ValidUntil      time.Time `json:"valid_until"`
  }
  ```
  
- [ ] **Task 4.1.3:** Implementar cálculo de distância e tempo
  - Usar fórmula Haversine para distância em linha reta
  - Tempo estimado: distância / velocidade média (30 km/h em cidade)
  - (Futuro: integrar com API de rotas como Google Maps/Mapbox)
  
- [ ] **Task 4.1.4:** Implementar lógica de precificação
  ```go
  type PricingConfig struct {
    BaseFare       float64 // R$ 5.00
    PricePerKm     float64 // R$ 2.50/km
    PricePerMinute float64 // R$ 0.50/min
  }
  
  func (s *FareService) Calculate(distKm, durationMin float64, surge float64) float64 {
    base := s.config.BaseFare
    distance := distKm * s.config.PricePerKm
    time := durationMin * s.config.PricePerMinute
    return (base + distance + time) * surge
  }
  ```
  
- [ ] **Task 4.1.5:** Implementar surge pricing (versão simplificada)
  - Surge = 1.0 (sem multiplicador) no MVP
  - (Futuro: calcular baseado em demanda vs oferta por região)
  
- [ ] **Task 4.1.6:** Persistir estimativa na tabela `fares`
  - Gerar UUID para fare_id
  - Salvar com valid_until = now() + 5 minutos
  - Retornar fare_id para ser usado na criação da corrida

---

### US-011: Solicitação de Corrida
**Como** passageiro  
**Quero** solicitar uma corrida  
**Para que** o sistema encontre um motorista para mim

#### Critérios de Aceite:
- Validação de fare_id válido e não expirado
- Endpoint cria registro na tabela `rides`
- Status inicial: REQUESTED
- Evento `ride.requested` é publicado no RabbitMQ

#### Tasks:
- [ ] **Task 4.2.1:** Implementar endpoint `POST /v1/rides`
  ```go
  type CreateRideRequest struct {
    RiderID        string  `json:"rider_id" binding:"required,uuid"`
    FareID         string  `json:"fare_id" binding:"required,uuid"`
    PickupLat      float64 `json:"pickup_lat" binding:"required"`
    PickupLng      float64 `json:"pickup_lng" binding:"required"`
    DestinationLat float64 `json:"destination_lat" binding:"required"`
    DestinationLng float64 `json:"destination_lng" binding:"required"`
  }
  
  type CreateRideResponse struct {
    RideID    string `json:"ride_id"`
    Status    string `json:"status"`
    CreatedAt time.Time `json:"created_at"`
  }
  ```
  
- [ ] **Task 4.2.2:** Validar fare_id
  - Buscar na tabela `fares` por ID
  - Verificar se valid_until > now()
  - Retornar erro 400 se fare expirado
  
- [ ] **Task 4.2.3:** Validar rider_id
  - Buscar usuário na tabela `users`
  - Verificar se user_type = 'RIDER'
  - Verificar se não tem corrida ativa (status IN ('REQUESTED', 'MATCHING', 'ACCEPTED', 'IN_PROGRESS'))
  
- [ ] **Task 4.2.4:** Criar registro na tabela `rides`
  ```go
  ride := &Ride{
    ID:                  uuid.New(),
    RiderID:             req.RiderID,
    FareID:              req.FareID,
    Status:              "REQUESTED",
    PickupLocation:      postgis.Point{Lat: req.PickupLat, Lng: req.PickupLng},
    DestinationLocation: postgis.Point{Lat: req.DestinationLat, Lng: req.DestinationLng},
  }
  ```
  
- [ ] **Task 4.2.5:** Publicar evento no RabbitMQ
  ```go
  event := RideRequestedEvent{
    EventID:       uuid.New().String(),
    CorrelationID: ctx.Value("request_id"),
    Timestamp:     time.Now(),
    Payload: RideRequestedPayload{
      RideID:        ride.ID.String(),
      RiderID:       ride.RiderID.String(),
      Pickup:        Location{Lat: req.PickupLat, Lng: req.PickupLng},
      Destination:   Location{Lat: req.DestinationLat, Lng: req.DestinationLng},
      EstimatedFare: fare.EstimatedTotal,
    },
  }
  
  err := s.publisher.Publish("rides.events", "ride.requested", event)
  ```
  
- [ ] **Task 4.2.6:** Implementar publisher RabbitMQ
  - Criar conexão com RabbitMQ
  - Implementar retry com exponential backoff
  - Confirmar publicação (publisher confirms)
  - Logar falhas de publicação

---

### US-012: Atualização de Status da Corrida
**Como** sistema  
**Quero** atualizar o status da corrida conforme o fluxo avança  
**Para que** passageiros e motoristas tenham visibilidade do progresso

#### Critérios de Aceite:
- Endpoint permite transições válidas de estado
- Transições inválidas retornam erro 400
- Cada mudança de status publica evento correspondente
- Auditoria automática via trigger `ride_updates`

#### Tasks:
- [ ] **Task 4.3.1:** Implementar endpoint `PATCH /v1/rides/:id/status`
  ```go
  type UpdateStatusRequest struct {
    NewStatus string            `json:"new_status" binding:"required"`
    Metadata  map[string]interface{} `json:"metadata"`
  }
  ```
  
- [ ] **Task 4.3.2:** Implementar máquina de estados
  ```go
  var validTransitions = map[string][]string{
    "REQUESTED":   {"MATCHING", "CANCELLED"},
    "MATCHING":    {"ACCEPTED", "CANCELLED"},
    "ACCEPTED":    {"IN_PROGRESS", "CANCELLED"},
    "IN_PROGRESS": {"COMPLETED", "CANCELLED"},
    "COMPLETED":   {},
    "CANCELLED":   {},
  }
  
  func (s *RideService) ValidateTransition(oldStatus, newStatus string) error {
    allowed := validTransitions[oldStatus]
    if !contains(allowed, newStatus) {
      return fmt.Errorf("invalid transition from %s to %s", oldStatus, newStatus)
    }
    return nil
  }
  ```
  
- [ ] **Task 4.3.3:** Atualizar registro no banco
  ```go
  func (r *RideRepository) UpdateStatus(ctx context.Context, rideID uuid.UUID, newStatus string) error {
    query := `
      UPDATE rides 
      SET status = $1,
          matched_at = CASE WHEN $1 = 'ACCEPTED' THEN NOW() ELSE matched_at END,
          started_at = CASE WHEN $1 = 'IN_PROGRESS' THEN NOW() ELSE started_at END,
          completed_at = CASE WHEN $1 = 'COMPLETED' THEN NOW() ELSE completed_at END,
          cancelled_at = CASE WHEN $1 = 'CANCELLED' THEN NOW() ELSE cancelled_at END
      WHERE id = $2
    `
    _, err := r.db.ExecContext(ctx, query, newStatus, rideID)
    return err
  }
  ```
  
- [ ] **Task 4.3.4:** Publicar eventos de mudança de estado
  - `ride.matched` quando status = ACCEPTED
  - `ride.started` quando status = IN_PROGRESS
  - `ride.completed` quando status = COMPLETED
  - `ride.cancelled` quando status = CANCELLED

---

### US-013: Aceite de Corrida pelo Motorista
**Como** motorista  
**Quero** aceitar uma oferta de corrida  
**Para que** eu possa iniciar o atendimento ao passageiro

#### Critérios de Aceite:
- Endpoint valida que motorista tem lock ativo
- Ride status muda de MATCHING → ACCEPTED
- Lock é liberado via Location Service
- Evento `matching.driver_accepted` é publicado

#### Tasks:
- [ ] **Task 4.4.1:** Implementar endpoint `PATCH /v1/rides/:id/accept`
  ```go
  type AcceptRideRequest struct {
    DriverID string `json:"driver_id" binding:"required,uuid"`
  }
  ```
  
- [ ] **Task 4.4.2:** Validar estado da corrida
  - Buscar ride por ID
  - Verificar se status = 'MATCHING'
  - Verificar se driver_id é NULL ou igual ao driver solicitante
  
- [ ] **Task 4.4.3:** Validar lock do motorista
  ```go
  func (s *RideService) ValidateDriverLock(ctx context.Context, driverID, rideID string) error {
    resp, err := s.locationClient.GetDriverStatus(ctx, driverID)
    if err != nil {
      return err
    }
    if !resp.IsLocked || resp.LockedForRideID != rideID {
      return errors.New("driver does not have valid lock for this ride")
    }
    return nil
  }
  ```
  
- [ ] **Task 4.4.4:** Atualizar ride no banco
  ```go
  UPDATE rides 
  SET status = 'ACCEPTED', 
      driver_id = $1,
      matched_at = NOW()
  WHERE id = $2 AND status = 'MATCHING'
  ```
  
- [ ] **Task 4.4.5:** Liberar lock via Location Service
  ```go
  err := s.locationClient.ReleaseLock(ctx, driverID, rideID)
  ```
  
- [ ] **Task 4.4.6:** Publicar evento `matching.driver_accepted`
  ```go
  event := DriverAcceptedEvent{
    RideID:     rideID,
    DriverID:   driverID,
    AcceptedAt: time.Now(),
  }
  s.publisher.Publish("matching.events", "matching.driver_accepted", event)
  ```
  
- [ ] **Task 4.4.7:** Publicar evento `ride.matched`
  ```go
  event := RideMatchedEvent{
    RideID:   rideID,
    RiderID:  ride.RiderID,
    DriverID: driverID,
  }
  s.publisher.Publish("rides.events", "ride.matched", event)
  ```

---

### US-014: Recusa de Corrida pelo Motorista
**Como** motorista  
**Quero** recusar uma oferta de corrida  
**Para que** o sistema tente encontrar outro motorista

#### Critérios de Aceite:
- Endpoint libera lock do motorista
- Evento `matching.driver_rejected` é publicado
- Matching Service consome evento e tenta próximo candidato
- Ride status permanece MATCHING

#### Tasks:
- [ ] **Task 4.5.1:** Implementar endpoint `PATCH /v1/rides/:id/reject`
  ```go
  type RejectRideRequest struct {
    DriverID string `json:"driver_id" binding:"required,uuid"`
    Reason   string `json:"reason,omitempty"`
  }
  ```
  
- [ ] **Task 4.5.2:** Validar estado da corrida
  - Verificar se status = 'MATCHING'
  - Verificar se motorista tem lock ativo
  
- [ ] **Task 4.5.3:** Liberar lock via Location Service
  ```go
  err := s.locationClient.ReleaseLock(ctx, driverID, rideID)
  ```
  
- [ ] **Task 4.5.4:** Registrar recusa em metadata
  ```go
  INSERT INTO ride_updates (ride_id, old_status, new_status, metadata)
  VALUES ($1, 'MATCHING', 'MATCHING', jsonb_build_object(
    'event', 'driver_rejected',
    'driver_id', $2,
    'reason', $3,
    'rejected_at', NOW()
  ))
  ```
  
- [ ] **Task 4.5.5:** Publicar evento `matching.driver_rejected`
  ```go
  event := DriverRejectedEvent{
    RideID:     rideID,
    DriverID:   driverID,
    RejectedAt: time.Now(),
    Reason:     req.Reason,
  }
  s.publisher.Publish("matching.events", "matching.driver_rejected", event)
  ```

---

### US-015: Cancelamento de Corrida
**Como** passageiro ou motorista  
**Quero** cancelar uma corrida  
**Para que** eu não seja cobrado/penalizado indevidamente

#### Critérios de Aceite:
- Cancelamento permitido apenas antes de IN_PROGRESS
- Após IN_PROGRESS, apenas motorista pode cancelar (casos excepcionais)
- Motivo do cancelamento é obrigatório
- Lock é liberado se existir

#### Tasks:
- [ ] **Task 4.6.1:** Implementar endpoint `PATCH /v1/rides/:id/cancel`
  ```go
  type CancelRideRequest struct {
    CancelledBy string `json:"cancelled_by" binding:"required,oneof=rider driver system"`
    Reason      string `json:"reason" binding:"required"`
  }
  ```
  
- [ ] **Task 4.6.2:** Validar permissões de cancelamento
  ```go
  func (s *RideService) CanCancel(ride *Ride, cancelledBy string) error {
    if ride.Status == "IN_PROGRESS" && cancelledBy == "rider" {
      return errors.New("rider cannot cancel ride in progress")
    }
    if ride.Status == "COMPLETED" || ride.Status == "CANCELLED" {
      return errors.New("ride already finished")
    }
    return nil
  }
  ```
  
- [ ] **Task 4.6.3:** Liberar lock se existir driver_id
  ```go
  if ride.DriverID != nil {
    s.locationClient.ReleaseLock(ctx, ride.DriverID.String(), rideID)
  }
  ```
  
- [ ] **Task 4.6.4:** Atualizar ride no banco
  ```go
  UPDATE rides 
  SET status = 'CANCELLED',
      cancelled_at = NOW(),
      cancellation_reason = $1
  WHERE id = $2
  ```
  
- [ ] **Task 4.6.5:** Publicar evento `ride.cancelled`
  ```go
  event := RideCancelledEvent{
    RideID:       rideID,
    RiderID:      ride.RiderID,
    DriverID:     ride.DriverID,
    CancelledBy:  req.CancelledBy,
    Reason:       req.Reason,
    CancelledAt:  time.Now(),
  }
  s.publisher.Publish("rides.events", "ride.cancelled", event)
  ```

---

## 5. Matching Service

### US-016: Consumir Eventos de Solicitação de Corrida
**Como** Matching Service  
**Quero** consumir eventos `ride.requested` do RabbitMQ  
**Para que** eu possa iniciar o processo de matching

#### Critérios de Aceite:
- Consumer conecta em fila dedicada `matching_service_queue`
- Mensagens são processadas com acknowledgment manual
- Retry automático em caso de falha transiente
- Dead letter queue para falhas permanentes

#### Tasks:
- [ ] **Task 5.1.1:** Setup do projeto Matching Service
  - Estrutura: `services/matching-service/`
  - Dependências: amqp, HTTP client para Location Service
  
- [ ] **Task 5.1.2:** Implementar consumer RabbitMQ
  ```go
  func (c *RabbitMQConsumer) Start(ctx context.Context) error {
    msgs, err := c.channel.Consume(
      "matching_service_queue", // queue
      "",                        // consumer
      false,                     // auto-ack
      false,                     // exclusive
      false,                     // no-local
      false,                     // no-wait
      nil,                       // args
    )
    
    for msg := range msgs {
      if err := c.handleMessage(ctx, msg); err != nil {
        msg.Nack(false, true) // requeue
      } else {
        msg.Ack(false)
      }
    }
  }
  ```
  
- [ ] **Task 5.1.3:** Deserializar evento `ride.requested`
  ```go
  type RideRequestedEvent struct {
    EventID       string    `json:"event_id"`
    CorrelationID string    `json:"correlation_id"`
    Timestamp     time.Time `json:"timestamp"`
    Payload       RideRequestedPayload `json:"payload"`
  }
  ```
  
- [ ] **Task 5.1.4:** Implementar handler do evento
  ```go
  func (h *MatchingHandler) HandleRideRequested(ctx context.Context, event RideRequestedEvent) error {
    // Buscar motoristas próximos
    // Tentar adquirir lock
    // Publicar match_candidate_found
    return nil
  }
  ```
  
- [ ] **Task 5.1.5:** Configurar retry policy
  - Max retries: 3
  - Backoff: exponential (1s, 2s, 4s)
  - Após 3 falhas: enviar para DLQ
  
- [ ] **Task 5.1.6:** Implementar graceful shutdown
  - Cancelar context ao receber SIGTERM
  - Aguardar processamento de mensagens em flight (timeout 30s)
  - Fechar conexão RabbitMQ

---

### US-017: Buscar e Rankear Motoristas Candidatos
**Como** Matching Service  
**Quero** buscar motoristas próximos e ordená-los por critérios  
**Para que** eu possa oferecer a corrida ao melhor candidato primeiro

#### Critérios de Aceite:
- Integração com Location Service via HTTP
- Ordenação por distância (primário) e rating (secundário - futuro)
- Filtro: apenas motoristas available e sem lock
- Fallback: aumentar raio de busca se não encontrar candidatos

#### Tasks:
- [ ] **Task 5.2.1:** Implementar client HTTP para Location Service
  ```go
  type LocationServiceClient struct {
    baseURL    string
    httpClient *http.Client
  }
  
  func (c *LocationServiceClient) FindNearbyDrivers(ctx context.Context, lat, lng, radius float64, limit int) ([]NearbyDriver, error) {
    url := fmt.Sprintf("%s/internal/drivers/nearby?lat=%f&lng=%f&radius=%f&limit=%d",
      c.baseURL, lat, lng, radius, limit)
    // ... make request
  }
  ```
  
- [ ] **Task 5.2.2:** Implementar lógica de busca com fallback
  ```go
  func (s *MatchingService) FindCandidates(ctx context.Context, pickup Location) ([]NearbyDriver, error) {
    radii := []float64{3000, 5000, 10000, 20000} // metros
    
    for _, radius := range radii {
      drivers, err := s.locationClient.FindNearbyDrivers(ctx, pickup.Lat, pickup.Lng, radius, 10)
      if err != nil {
        return nil, err
      }
      if len(drivers) > 0 {
        return drivers, nil
      }
    }
    return nil, errors.New("no drivers found in area")
  }
  ```
  
- [ ] **Task 5.2.3:** Implementar ordenação de candidatos
  ```go
  func (s *MatchingService) RankCandidates(drivers []NearbyDriver) []NearbyDriver {
    sort.Slice(drivers, func(i, j int) bool {
      // Ordenar por distância (crescente)
      return drivers[i].DistanceMeters < drivers[j].DistanceMeters
      // Futuro: considerar rating, acceptance rate, etc
    })
    return drivers
  }
  ```

---

### US-018: Adquirir Lock e Oferecer Corrida
**Como** Matching Service  
**Quero** adquirir lock exclusivo no motorista e enviar oferta  
**Para que** apenas uma corrida seja oferecida por vez ao motorista

#### Critérios de Aceite:
- Tentativa de lock em cada candidato sequencialmente
- Se lock falhar, tentar próximo candidato
- Publicar evento `matching.candidate_found` após lock bem-sucedido
- Registrar tentativas e latência

#### Tasks:
- [ ] **Task 5.3.1:** Implementar método de aquisição de lock
  ```go
  func (c *LocationServiceClient) AcquireLock(ctx context.Context, driverID, rideID string, ttl int) (*LockResponse, error) {
    url := fmt.Sprintf("%s/internal/drivers/%s/lock", c.baseURL, driverID)
    body := LockRequest{RideID: rideID, TTLSeconds: ttl}
    // ... POST request
  }
  ```
  
- [ ] **Task 5.3.2:** Implementar loop de tentativas
  ```go
  func (s *MatchingService) TryLockCandidates(ctx context.Context, rideID string, candidates []NearbyDriver) (*NearbyDriver, error) {
    for _, candidate := range candidates {
      resp, err := s.locationClient.AcquireLock(ctx, candidate.DriverID, rideID, 15)
      if err != nil {
        log.Warn("failed to acquire lock", "driver", candidate.DriverID, "error", err)
        continue
      }
      if resp.Locked {
        return &candidate, nil
      }
    }
    return nil, errors.New("no available drivers to lock")
  }
  ```
  
- [ ] **Task 5.3.3:** Publicar evento `matching.candidate_found`
  ```go
  func (s *MatchingService) PublishCandidateFound(ctx context.Context, rideID string, driver NearbyDriver) error {
    event := CandidateFoundEvent{
      EventID: uuid.New().String(),
      Payload: CandidateFoundPayload{
        RideID:              rideID,
        DriverID:            driver.DriverID,
        ExpiresAt:           time.Now().Add(20 * time.Second),
        DistanceToPickupMeters: driver.DistanceMeters,
      },
    }
    return s.publisher.Publish("matching.events", "matching.candidate_found", event)
  }
  ```
  
- [ ] **Task 5.3.4:** Registrar métricas
  - Histogram: tempo para encontrar match
  - Counter: tentativas de lock (sucesso/falha)
  - Counter: número de candidatos tentados por ride

---

### US-019: Gerenciar Timeouts e Retries
**Como** Matching Service  
**Quero** gerenciar timeouts de resposta do motorista e tentar próximo candidato  
**Para que** a solicitação do passageiro não fique travada

#### Critérios de Aceite:
- Integração com Temporal.io para workflows com timeout
- Timeout de 20 segundos para resposta do motorista
- Retry automático com próximo candidato
- Limite de 10 tentativas antes de desistir

#### Tasks:
- [ ] **Task 5.4.1:** Setup do Temporal.io
  - Instalar Temporal Server via Docker Compose
  - Configurar worker do Matching Service
  - Criar namespace: `ride-sharing`
  
- [ ] **Task 5.4.2:** Implementar Workflow de Matching
  ```go
  func MatchingWorkflow(ctx workflow.Context, rideID string, pickup Location) error {
    ctx = workflow.WithActivityOptions(ctx, workflow.ActivityOptions{
      StartToCloseTimeout: 30 * time.Second,
    })
    
    var result MatchingResult
    err := workflow.ExecuteActivity(ctx, FindAndLockDriver, rideID, pickup).Get(ctx, &result)
    if err != nil {
      return err
    }
    
    // Aguardar aceite do motorista com timeout
    selector := workflow.NewSelector(ctx)
    
    timeoutFuture := workflow.NewTimer(ctx, 20*time.Second)
    selector.AddFuture(timeoutFuture, func(f workflow.Future) {
      // Timeout - tentar próximo candidato
      workflow.ExecuteActivity(ctx, ReleaseLock, result.DriverID, rideID)
      workflow.ExecuteActivity(ctx, RetryMatching, rideID)
    })
    
    acceptSignal := workflow.GetSignalChannel(ctx, "driver_accepted")
    selector.AddReceive(acceptSignal, func(c workflow.ReceiveChannel, more bool) {
      // Motorista aceitou - workflow completo
      workflow.CompleteWorkflow(ctx, result)
    })
    
    selector.Select(ctx)
    return nil
  }
  ```
  
- [ ] **Task 5.4.3:** Implementar Activities
  ```go
  func FindAndLockDriver(ctx context.Context, rideID string, pickup Location) (*MatchingResult, error)
  func ReleaseLock(ctx context.Context, driverID, rideID string) error
  func RetryMatching(ctx context.Context, rideID string) error
  ```
  
- [ ] **Task 5.4.4:** Consumir eventos de aceite/rejeição
  - Consumer para `matching.driver_accepted` → enviar signal para workflow
  - Consumer para `matching.driver_rejected` → enviar signal para workflow
  - Consumer para `matching.driver_timeout` → cancelar timer
  
- [ ] **Task 5.4.5:** Implementar limite de tentativas
  ```go
  func MatchingWorkflow(ctx workflow.Context, input MatchingInput) error {
    maxAttempts := 10
    for attempt := 1; attempt <= maxAttempts; attempt++ {
      // ... try match
      if matched {
        return nil
      }
      workflow.Sleep(ctx, 5*time.Second) // backoff entre tentativas
    }
    
    // Esgotar tentativas - publicar evento de falha
    workflow.ExecuteActivity(ctx, PublishMatchingFailed, input.RideID)
    return errors.New("matching failed after max attempts")
  }
  ```

---

### US-020: Notificar Falha de Matching
**Como** sistema  
**Quero** notificar passageiro quando não encontrar motorista  
**Para que** ele saiba que precisa tentar novamente

#### Critérios de Aceite:
- Evento `matching.no_drivers_available` é publicado após 5 minutos
- Ride status muda para CANCELLED
- Passageiro recebe notificação com sugestões

#### Tasks:
- [ ] **Task 5.5.1:** Implementar Activity de falha
  ```go
  func PublishMatchingFailed(ctx context.Context, rideID string) error {
    event := MatchingFailedEvent{
      RideID:           rideID,
      Reason:           "no_drivers_available",
      TotalAttempts:    10,
      TimeoutAt:        time.Now(),
    }
    return publisher.Publish("matching.events", "matching.no_drivers_available", event)
  }
  ```
  
- [ ] **Task 5.5.2:** Ride Service consome evento e cancela ride
  - Consumer para `matching.no_drivers_available`
  - UPDATE rides SET status = 'CANCELLED', cancellation_reason = 'no_drivers_available'
  - Publicar evento `ride.cancelled`

---

## 6. Notification Service

### US-021: Gerenciar Conexões WebSocket de Clientes
**Como** passageiro ou motorista  
**Quero** manter conexão WebSocket persistente  
**Para que** eu receba notificações em tempo real

#### Critérios de Aceite:
- Endpoint WebSocket aceita conexões autenticadas
- Conexões são mapeadas por user_id/driver_id
- Heartbeat para detectar desconexões
- Suporte a 50k conexões simultâneas por instância

#### Tasks:
- [ ] **Task 6.1.1:** Setup do projeto Notification Service
  - Estrutura: `services/notification-service/`
  - Dependências: gorilla/websocket, amqp
  
- [ ] **Task 6.1.2:** Implementar WebSocket handler
  ```go
  func (h *WSHandler) HandleConnection(c *gin.Context) {
    userID := c.Query("user_id")
    userType := c.Query("user_type") // rider | driver
    
    conn, err := upgrader.Upgrade(c.Writer, c.Request, nil)
    if err != nil {
      return
    }
    
    client := &Client{
      ID:       userID,
      Type:     userType,
      Conn:     conn,
      SendChan: make(chan []byte, 256),
    }
    
    h.hub.Register(client)
    
    go client.ReadPump()
    go client.WritePump()
  }
  ```
  
- [ ] **Task 6.1.3:** Implementar Hub para gerenciar conexões
  ```go
  type Hub struct {
    clients    map[string]*Client // key: userID
    broadcast  chan Message
    register   chan *Client
    unregister chan *Client
  }
  
  func (h *Hub) Run() {
    for {
      select {
      case client := <-h.register:
        h.clients[client.ID] = client
      case client := <-h.unregister:
        delete(h.clients, client.ID)
        close(client.SendChan)
      case message := <-h.broadcast:
        if client, ok := h.clients[message.TargetUserID]; ok {
          client.SendChan <- message.Data
        }
      }
    }
  }
  ```
  
- [ ] **Task 6.1.4:** Implementar heartbeat
  - Cliente envia ping a cada 30 segundos
  - Servidor responde com pong
  - Desconectar se não receber ping por 60 segundos
  
- [ ] **Task 6.1.5:** Implementar ReadPump e WritePump
  ```go
  func (c *Client) ReadPump() {
    defer c.Conn.Close()
    for {
      _, message, err := c.Conn.ReadMessage()
      if err != nil {
        hub.unregister <- c
        break
      }
      // Handle incoming messages (pings, acks, etc)
    }
  }
  
  func (c *Client) WritePump() {
    defer c.Conn.Close()
    for {
      select {
      case message := <-c.SendChan:
        c.Conn.WriteMessage(websocket.TextMessage, message)
      }
    }
  }
  ```

---

### US-022: Consumir Eventos e Rotear Notificações
**Como** Notification Service  
**Quero** consumir eventos do RabbitMQ e rotear para clientes conectados  
**Para que** usuários recebam atualizações em tempo real

#### Critérios de Aceite:
- Consumers para múltiplas filas (rides, matching, location)
- Roteamento baseado em user_id/driver_id
- Notificações formatadas por tipo de evento
- Fallback: salvar notificação se cliente offline

#### Tasks:
- [ ] **Task 6.2.1:** Implementar consumer para `notification_rides_queue`
  - Binding: `ride.matched`, `ride.started`, `ride.completed`, `ride.cancelled`
  - Handler: extrair rider_id e driver_id, enviar notificação para ambos
  
- [ ] **Task 6.2.2:** Implementar consumer para `notification_matching_queue`
  - Binding: `matching.candidate_found`
  - Handler: extrair driver_id, enviar oferta de corrida
  
- [ ] **Task 6.2.3:** Implementar consumer para `tracking_realtime_queue`
  - Binding: `location.driver.updated`
  - Handler: se motorista está em corrida ativa, enviar posição para passageiro
  
- [ ] **Task 6.2.4:** Implementar roteador de mensagens
  ```go
  func (s *NotificationService) RouteMessage(ctx context.Context, event Event) error {
    notification := s.buildNotification(event)
    
    for _, targetUserID := range event.GetTargetUserIDs() {
      if client, ok := s.hub.clients[targetUserID]; ok {
        client.SendChan <- notification
      } else {
        // Cliente offline - salvar em fila de notificações pendentes
        s.saveOfflineNotification(targetUserID, notification)
      }
    }
    return nil
  }
  ```
  
- [ ] **Task 6.2.5:** Implementar formatadores de notificação por tipo
  ```go
  type NotificationFormatter interface {
    Format(event Event) (*Notification, error)
  }
  
  type RideMatchedFormatter struct{}
  func (f *RideMatchedFormatter) Format(event Event) (*Notification, error) {
    return &Notification{
      Type:    "ride_matched",
      Title:   "Motorista encontrado!",
      Message: fmt.Sprintf("%s está a caminho", event.DriverName),
      Data:    event.Payload,
    }, nil
  }
  ```

---

### US-023: Enviar Ofertas de Corrida para Motoristas
**Como** Notification Service  
**Quero** enviar detalhes da corrida para o motorista selecionado  
**Para que** ele possa decidir aceitar ou recusar

#### Critérios de Aceite:
- Mensagem contém: pickup location, destination, estimated fare, distance
- Timer visível no app: 20 segundos para responder
- Botões: Aceitar / Recusar

#### Tasks:
- [ ] **Task 6.3.1:** Implementar handler de `matching.candidate_found`
  ```go
  func (h *MatchingEventHandler) HandleCandidateFound(event CandidateFoundEvent) error {
    notification := &Notification{
      Type:      "ride_offer",
      Title:     "Nova corrida disponível!",
      Message:   fmt.Sprintf("R$ %.2f - %.1f km de distância", event.EstimatedFare, event.DistanceMeters/1000),
      ExpiresAt: event.ExpiresAt,
      Data: map[string]interface{}{
        "ride_id":              event.RideID,
        "pickup_lat":           event.PickupLat,
        "pickup_lng":           event.PickupLng,
        "destination_lat":      event.DestinationLat,
        "destination_lng":      event.DestinationLng,
        "estimated_fare":       event.EstimatedFare,
        "distance_to_pickup":   event.DistanceToPickupMeters,
      },
    }
    
    return h.hub.SendToUser(event.DriverID, notification)
  }
  ```
  
- [ ] **Task 6.3.2:** Implementar timer no cliente (responsabilidade do app)
  - App exibe countdown de 20 segundos
  - Após timeout, esconder oferta automaticamente

---

### US-024: Tracking de Localização em Tempo Real
**Como** passageiro  
**Quero** ver a localização do motorista em tempo real  
**Para que** eu saiba quando ele chegará

#### Critérios de Aceite:
- Atualização de posição a cada 3 segundos
- Apenas para corridas ativas (ACCEPTED, IN_PROGRESS)
- Mensagens otimizadas (apenas lat/lng, sem payload extra)

#### Tasks:
- [ ] **Task 6.4.1:** Implementar consumer de `location.driver.updated`
  - Filtrar eventos: apenas motoristas em corrida ativa
  - Buscar ride_id associado ao driver_id
  - Enviar atualização para rider_id da corrida
  
- [ ] **Task 6.4.2:** Implementar cache de corridas ativas
  ```go
  type ActiveRidesCache struct {
    rides map[string]string // key: driver_id, value: rider_id
    mu    sync.RWMutex
  }
  
  func (c *ActiveRidesCache) GetRiderForDriver(driverID string) (string, bool) {
    c.mu.RLock()
    defer c.mu.RUnlock()
    riderID, ok := c.rides[driverID]
    return riderID, ok
  }
  ```
  
- [ ] **Task 6.4.3:** Atualizar cache ao consumir eventos de ride
  - `ride.matched` → adicionar mapping driver_id → rider_id
  - `ride.completed` ou `ride.cancelled` → remover mapping
  
- [ ] **Task 6.4.4:** Implementar throttling de atualizações
  - Limitar a 1 update a cada 3 segundos por corrida
  - Evitar flood de mensagens se Location Service enviar pings mais frequentes

---

## 7. API Gateway e gRPC

### US-025: Estrutura de Protos Compartilhados
**Como** desenvolvedor full-stack
**Quero** definições Protocol Buffers compartilhadas entre backend e frontend
**Para que** o frontend tenha tipagem automática de requests/responses

#### Critérios de Aceite:
- Estrutura de diretórios proto bem organizada
- Arquivos .proto com google/api/annotations para mapeamento HTTP
- Geração automática de código Go e TypeScript/JavaScript
- Versionamento de APIs (v1, v2)
- Documentação automática via OpenAPI/Swagger

#### Tasks:
- [ ] **Task 7.1.1:** Criar estrutura de diretórios proto
  ```
  proto/
  ├── google/
  │   └── api/
  │       ├── annotations.proto
  │       ├── http.proto
  │       └── field_behavior.proto
  ├── ridesharing/
  │   ├── v1/
  │   │   ├── common.proto           # Tipos compartilhados
  │   │   ├── user.proto
  │   │   ├── ride.proto
  │   │   ├── location.proto
  │   │   ├── notification.proto
  │   │   └── auth.proto
  │   └── v2/                         # Futuras versões
  └── buf.yaml                        # Buf schema registry config
  ```

- [ ] **Task 7.1.2:** Criar proto de tipos comuns
  ```protobuf
  // proto/ridesharing/v1/common.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/protobuf/timestamp.proto";
  import "google/api/field_behavior.proto";

  // Location representa coordenadas geográficas
  message Location {
    double latitude = 1 [(google.api.field_behavior) = REQUIRED];
    double longitude = 2 [(google.api.field_behavior) = REQUIRED];
  }

  // Money representa valores monetários
  message Money {
    string currency_code = 1; // ISO 4217 (e.g., "BRL")
    int64 units = 2;           // Parte inteira (ex: 10)
    int32 nanos = 3;           // Parte decimal (ex: 500000000 = 0.50)
  }

  // Pagination para listagem
  message PageRequest {
    int32 page_size = 1;      // Máximo de resultados por página
    string page_token = 2;    // Token da próxima página
  }

  message PageResponse {
    string next_page_token = 1;
    int32 total_count = 2;
  }

  // Status padrão de operações
  enum OperationStatus {
    OPERATION_STATUS_UNSPECIFIED = 0;
    OPERATION_STATUS_PENDING = 1;
    OPERATION_STATUS_IN_PROGRESS = 2;
    OPERATION_STATUS_COMPLETED = 3;
    OPERATION_STATUS_FAILED = 4;
    OPERATION_STATUS_CANCELLED = 5;
  }

  // ErrorDetails para respostas de erro enriquecidas
  message ErrorDetail {
    string code = 1;
    string message = 2;
    map<string, string> metadata = 3;
  }
  ```

- [ ] **Task 7.1.3:** Criar proto do User Service
  ```protobuf
  // proto/ridesharing/v1/user.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/api/annotations.proto";
  import "google/api/field_behavior.proto";
  import "google/protobuf/timestamp.proto";
  import "google/protobuf/empty.proto";
  import "ridesharing/v1/common.proto";

  service UserService {
    // CreateUser cria um novo usuário (rider ou driver)
    rpc CreateUser(CreateUserRequest) returns (User) {
      option (google.api.http) = {
        post: "/v1/users"
        body: "*"
      };
    }

    // GetUser busca usuário por ID
    rpc GetUser(GetUserRequest) returns (User) {
      option (google.api.http) = {
        get: "/v1/users/{user_id}"
      };
    }

    // UpdateUser atualiza dados do usuário
    rpc UpdateUser(UpdateUserRequest) returns (User) {
      option (google.api.http) = {
        patch: "/v1/users/{user_id}"
        body: "*"
      };
    }

    // DeleteUser remove usuário (soft delete)
    rpc DeleteUser(DeleteUserRequest) returns (google.protobuf.Empty) {
      option (google.api.http) = {
        delete: "/v1/users/{user_id}"
      };
    }

    // ListUsers lista usuários com filtros
    rpc ListUsers(ListUsersRequest) returns (ListUsersResponse) {
      option (google.api.http) = {
        get: "/v1/users"
      };
    }
  }

  enum UserType {
    USER_TYPE_UNSPECIFIED = 0;
    USER_TYPE_RIDER = 1;
    USER_TYPE_DRIVER = 2;
  }

  message User {
    string user_id = 1;
    string full_name = 2;
    string email = 3;
    UserType user_type = 4;
    google.protobuf.Timestamp created_at = 5;
    google.protobuf.Timestamp updated_at = 6;
  }

  message CreateUserRequest {
    string full_name = 1 [(google.api.field_behavior) = REQUIRED];
    string email = 2 [(google.api.field_behavior) = REQUIRED];
    UserType user_type = 3 [(google.api.field_behavior) = REQUIRED];
  }

  message GetUserRequest {
    string user_id = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message UpdateUserRequest {
    string user_id = 1 [(google.api.field_behavior) = REQUIRED];
    string full_name = 2;
    string email = 3;
  }

  message DeleteUserRequest {
    string user_id = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message ListUsersRequest {
    UserType user_type = 1;
    PageRequest page = 2;
  }

  message ListUsersResponse {
    repeated User users = 1;
    PageResponse page = 2;
  }
  ```

- [ ] **Task 7.1.4:** Criar proto do Ride Service
  ```protobuf
  // proto/ridesharing/v1/ride.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/api/annotations.proto";
  import "google/api/field_behavior.proto";
  import "google/protobuf/timestamp.proto";
  import "ridesharing/v1/common.proto";

  service RideService {
    // EstimateFare calcula estimativa de tarifa
    rpc EstimateFare(EstimateFareRequest) returns (Fare) {
      option (google.api.http) = {
        post: "/v1/fares/estimate"
        body: "*"
      };
    }

    // CreateRide solicita uma nova corrida
    rpc CreateRide(CreateRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides"
        body: "*"
      };
    }

    // GetRide busca corrida por ID
    rpc GetRide(GetRideRequest) returns (Ride) {
      option (google.api.http) = {
        get: "/v1/rides/{ride_id}"
      };
    }

    // AcceptRide motorista aceita corrida
    rpc AcceptRide(AcceptRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides/{ride_id}/accept"
        body: "*"
      };
    }

    // RejectRide motorista rejeita corrida
    rpc RejectRide(RejectRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides/{ride_id}/reject"
        body: "*"
      };
    }

    // StartRide inicia corrida
    rpc StartRide(StartRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides/{ride_id}/start"
        body: "*"
      };
    }

    // CompleteRide finaliza corrida
    rpc CompleteRide(CompleteRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides/{ride_id}/complete"
        body: "*"
      };
    }

    // CancelRide cancela corrida
    rpc CancelRide(CancelRideRequest) returns (Ride) {
      option (google.api.http) = {
        post: "/v1/rides/{ride_id}/cancel"
        body: "*"
      };
    }

    // ListRides lista corridas do usuário
    rpc ListRides(ListRidesRequest) returns (ListRidesResponse) {
      option (google.api.http) = {
        get: "/v1/rides"
      };
    }
  }

  enum RideStatus {
    RIDE_STATUS_UNSPECIFIED = 0;
    RIDE_STATUS_REQUESTED = 1;
    RIDE_STATUS_MATCHING = 2;
    RIDE_STATUS_ACCEPTED = 3;
    RIDE_STATUS_IN_PROGRESS = 4;
    RIDE_STATUS_COMPLETED = 5;
    RIDE_STATUS_CANCELLED = 6;
  }

  message Fare {
    string fare_id = 1;
    Location pickup_location = 2;
    Location destination_location = 3;
    int32 distance_meters = 4;
    int32 duration_seconds = 5;
    Money base_fare = 6;
    Money distance_fare = 7;
    Money time_fare = 8;
    double surge_multiplier = 9;
    Money estimated_total = 10;
    google.protobuf.Timestamp valid_until = 11;
  }

  message Ride {
    string ride_id = 1;
    string rider_id = 2;
    string driver_id = 3;
    string fare_id = 4;
    RideStatus status = 5;
    Location pickup_location = 6;
    Location destination_location = 7;
    Money estimated_fare = 8;
    Money final_fare = 9;
    google.protobuf.Timestamp requested_at = 10;
    google.protobuf.Timestamp matched_at = 11;
    google.protobuf.Timestamp started_at = 12;
    google.protobuf.Timestamp completed_at = 13;
    google.protobuf.Timestamp cancelled_at = 14;
    string cancellation_reason = 15;
  }

  message EstimateFareRequest {
    Location pickup_location = 1 [(google.api.field_behavior) = REQUIRED];
    Location destination_location = 2 [(google.api.field_behavior) = REQUIRED];
  }

  message CreateRideRequest {
    string rider_id = 1 [(google.api.field_behavior) = REQUIRED];
    string fare_id = 2 [(google.api.field_behavior) = REQUIRED];
    Location pickup_location = 3 [(google.api.field_behavior) = REQUIRED];
    Location destination_location = 4 [(google.api.field_behavior) = REQUIRED];
  }

  message GetRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message AcceptRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
    string driver_id = 2 [(google.api.field_behavior) = REQUIRED];
  }

  message RejectRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
    string driver_id = 2 [(google.api.field_behavior) = REQUIRED];
    string reason = 3;
  }

  message StartRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message CompleteRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message CancelRideRequest {
    string ride_id = 1 [(google.api.field_behavior) = REQUIRED];
    string cancelled_by = 2;
    string reason = 3 [(google.api.field_behavior) = REQUIRED];
  }

  message ListRidesRequest {
    string user_id = 1;
    RideStatus status = 2;
    PageRequest page = 3;
  }

  message ListRidesResponse {
    repeated Ride rides = 1;
    PageResponse page = 2;
  }
  ```

- [ ] **Task 7.1.5:** Criar proto do Location Service
  ```protobuf
  // proto/ridesharing/v1/location.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/api/annotations.proto";
  import "google/api/field_behavior.proto";
  import "google/protobuf/timestamp.proto";
  import "ridesharing/v1/common.proto";

  service LocationService {
    // UpdateDriverLocation atualiza localização do motorista (HTTP polling fallback)
    rpc UpdateDriverLocation(UpdateDriverLocationRequest) returns (UpdateDriverLocationResponse) {
      option (google.api.http) = {
        post: "/v1/location/drivers/{driver_id}"
        body: "*"
      };
    }

    // GetNearbyDrivers busca motoristas próximos (internal)
    rpc GetNearbyDrivers(GetNearbyDriversRequest) returns (GetNearbyDriversResponse) {
      option (google.api.http) = {
        get: "/v1/location/drivers/nearby"
      };
    }

    // AcquireDriverLock adquire lock exclusivo em motorista (internal)
    rpc AcquireDriverLock(AcquireDriverLockRequest) returns (AcquireDriverLockResponse) {
      option (google.api.http) = {
        post: "/v1/location/drivers/{driver_id}/lock"
        body: "*"
      };
    }

    // ReleaseDriverLock libera lock do motorista (internal)
    rpc ReleaseDriverLock(ReleaseDriverLockRequest) returns (ReleaseDriverLockResponse) {
      option (google.api.http) = {
        delete: "/v1/location/drivers/{driver_id}/lock"
      };
    }

    // GetDriverStatus obtém status atual do motorista (internal)
    rpc GetDriverStatus(GetDriverStatusRequest) returns (DriverStatus) {
      option (google.api.http) = {
        get: "/v1/location/drivers/{driver_id}/status"
      };
    }
  }

  enum DriverAvailability {
    DRIVER_AVAILABILITY_UNSPECIFIED = 0;
    DRIVER_AVAILABILITY_AVAILABLE = 1;
    DRIVER_AVAILABILITY_BUSY = 2;
    DRIVER_AVAILABILITY_OFFLINE = 3;
  }

  message DriverLocation {
    string driver_id = 1;
    Location location = 2;
    double heading = 3;
    double speed_kmh = 4;
    google.protobuf.Timestamp timestamp = 5;
  }

  message DriverStatus {
    string driver_id = 1;
    DriverAvailability availability = 2;
    bool is_locked = 3;
    string locked_for_ride_id = 4;
    google.protobuf.Timestamp last_ping = 5;
    string current_ride_id = 6;
  }

  message NearbyDriver {
    string driver_id = 1;
    double distance_meters = 2;
    google.protobuf.Timestamp last_seen = 3;
    DriverAvailability availability = 4;
  }

  message UpdateDriverLocationRequest {
    string driver_id = 1 [(google.api.field_behavior) = REQUIRED];
    Location location = 2 [(google.api.field_behavior) = REQUIRED];
    double heading = 3;
    double speed_kmh = 4;
  }

  message UpdateDriverLocationResponse {
    bool success = 1;
  }

  message GetNearbyDriversRequest {
    Location location = 1 [(google.api.field_behavior) = REQUIRED];
    double radius_meters = 2 [(google.api.field_behavior) = REQUIRED];
    int32 limit = 3;
  }

  message GetNearbyDriversResponse {
    repeated NearbyDriver drivers = 1;
  }

  message AcquireDriverLockRequest {
    string driver_id = 1 [(google.api.field_behavior) = REQUIRED];
    string ride_id = 2 [(google.api.field_behavior) = REQUIRED];
    int32 ttl_seconds = 3;
  }

  message AcquireDriverLockResponse {
    bool locked = 1;
    google.protobuf.Timestamp expires_at = 2;
    string reason = 3;
  }

  message ReleaseDriverLockRequest {
    string driver_id = 1 [(google.api.field_behavior) = REQUIRED];
    string ride_id = 2 [(google.api.field_behavior) = REQUIRED];
  }

  message ReleaseDriverLockResponse {
    bool released = 1;
  }

  message GetDriverStatusRequest {
    string driver_id = 1 [(google.api.field_behavior) = REQUIRED];
  }
  ```

- [ ] **Task 7.1.6:** Criar proto do Notification Service (WebSocket)
  ```protobuf
  // proto/ridesharing/v1/notification.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/protobuf/timestamp.proto";
  import "google/protobuf/struct.proto";
  import "ridesharing/v1/common.proto";

  // WebSocket messages (não usa HTTP annotations)
  service NotificationService {
    // StreamNotifications mantém stream bidirecional de notificações
    // Frontend usa este proto para tipagem do WebSocket
    rpc StreamNotifications(stream ClientMessage) returns (stream ServerMessage);
  }

  // Mensagens do Cliente -> Servidor
  message ClientMessage {
    oneof message {
      PingMessage ping = 1;
      AckMessage ack = 2;
      SubscribeMessage subscribe = 3;
      UnsubscribeMessage unsubscribe = 4;
    }
  }

  message PingMessage {
    google.protobuf.Timestamp timestamp = 1;
  }

  message AckMessage {
    string message_id = 1;
  }

  message SubscribeMessage {
    repeated string topics = 1; // e.g., "rides.*", "location.driver.123"
  }

  message UnsubscribeMessage {
    repeated string topics = 1;
  }

  // Mensagens do Servidor -> Cliente
  message ServerMessage {
    oneof message {
      PongMessage pong = 1;
      NotificationMessage notification = 2;
      ErrorMessage error = 3;
    }
  }

  message PongMessage {
    google.protobuf.Timestamp timestamp = 1;
  }

  message NotificationMessage {
    string message_id = 1;
    NotificationType type = 2;
    string title = 3;
    string body = 4;
    google.protobuf.Timestamp timestamp = 5;
    google.protobuf.Struct data = 6; // Payload específico do tipo
  }

  message ErrorMessage {
    string code = 1;
    string message = 2;
  }

  enum NotificationType {
    NOTIFICATION_TYPE_UNSPECIFIED = 0;
    NOTIFICATION_TYPE_RIDE_MATCHED = 1;
    NOTIFICATION_TYPE_RIDE_OFFER = 2;
    NOTIFICATION_TYPE_RIDE_STARTED = 3;
    NOTIFICATION_TYPE_RIDE_COMPLETED = 4;
    NOTIFICATION_TYPE_RIDE_CANCELLED = 5;
    NOTIFICATION_TYPE_DRIVER_LOCATION_UPDATE = 6;
    NOTIFICATION_TYPE_MATCHING_FAILED = 7;
  }
  ```

- [ ] **Task 7.1.7:** Criar proto do Auth Service
  ```protobuf
  // proto/ridesharing/v1/auth.proto
  syntax = "proto3";

  package ridesharing.v1;

  option go_package = "github.com/org/ride-sharing/gen/go/ridesharing/v1;ridesharingv1";

  import "google/api/annotations.proto";
  import "google/api/field_behavior.proto";
  import "google/protobuf/timestamp.proto";
  import "ridesharing/v1/user.proto";

  service AuthService {
    // Login autentica usuário e retorna JWT
    rpc Login(LoginRequest) returns (LoginResponse) {
      option (google.api.http) = {
        post: "/v1/auth/login"
        body: "*"
      };
    }

    // RefreshToken renova JWT expirado
    rpc RefreshToken(RefreshTokenRequest) returns (LoginResponse) {
      option (google.api.http) = {
        post: "/v1/auth/refresh"
        body: "*"
      };
    }

    // Logout invalida tokens
    rpc Logout(LogoutRequest) returns (LogoutResponse) {
      option (google.api.http) = {
        post: "/v1/auth/logout"
        body: "*"
      };
    }

    // ValidateToken valida JWT (internal)
    rpc ValidateToken(ValidateTokenRequest) returns (ValidateTokenResponse) {
      option (google.api.http) = {
        post: "/v1/auth/validate"
        body: "*"
      };
    }
  }

  message LoginRequest {
    string email = 1 [(google.api.field_behavior) = REQUIRED];
    string password = 2 [(google.api.field_behavior) = REQUIRED];
  }

  message LoginResponse {
    string access_token = 1;
    string refresh_token = 2;
    google.protobuf.Timestamp expires_at = 3;
    UserType user_type = 4;
    string user_id = 5;
  }

  message RefreshTokenRequest {
    string refresh_token = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message LogoutRequest {
    string access_token = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message LogoutResponse {
    bool success = 1;
  }

  message ValidateTokenRequest {
    string access_token = 1 [(google.api.field_behavior) = REQUIRED];
  }

  message ValidateTokenResponse {
    bool valid = 1;
    string user_id = 2;
    UserType user_type = 3;
    google.protobuf.Timestamp expires_at = 4;
  }
  ```

- [ ] **Task 7.1.8:** Configurar buf.yaml para validação e geração
  ```yaml
  # proto/buf.yaml
  version: v1
  breaking:
    use:
      - FILE
  lint:
    use:
      - DEFAULT
      - COMMENTS
      - FILE_LOWER_SNAKE_CASE
      - PACKAGE_VERSION_SUFFIX
    except:
      - PACKAGE_DIRECTORY_MATCH
  ```

- [ ] **Task 7.1.9:** Criar Makefile para geração de código
  ```makefile
  # Makefile (adicionar)

  .PHONY: proto-gen proto-lint proto-breaking

  # Gerar código Go e TypeScript dos protos
  proto-gen:
  	@echo "Generating Go code from protos..."
  	buf generate proto
  	@echo "Generating TypeScript code from protos..."
  	protoc -I proto \
  		--plugin=protoc-gen-ts=./node_modules/.bin/protoc-gen-ts \
  		--ts_out=frontend/src/generated \
  		proto/ridesharing/v1/*.proto

  # Validar protos
  proto-lint:
  	buf lint proto

  # Verificar breaking changes
  proto-breaking:
  	buf breaking proto --against '.git#branch=main'
  ```

- [ ] **Task 7.1.10:** Criar buf.gen.yaml para configuração de geração
  ```yaml
  # proto/buf.gen.yaml
  version: v1
  managed:
    enabled: true
    go_package_prefix:
      default: github.com/org/ride-sharing/gen/go
  plugins:
    # Gerar código Go
    - plugin: buf.build/protocolbuffers/go
      out: ../gen/go
      opt:
        - paths=source_relative

    # Gerar gRPC Go
    - plugin: buf.build/grpc/go
      out: ../gen/go
      opt:
        - paths=source_relative

    # Gerar grpc-gateway (HTTP->gRPC)
    - plugin: buf.build/grpc-ecosystem/gateway
      out: ../gen/go
      opt:
        - paths=source_relative
        - generate_unbound_methods=true

    # Gerar OpenAPI/Swagger
    - plugin: buf.build/grpc-ecosystem/openapiv2
      out: ../gen/openapi
      opt:
        - allow_merge=true
        - merge_file_name=api
  ```

---

### US-026: Implementar API Gateway em Go com gRPC-Gateway
**Como** desenvolvedor
**Quero** um API Gateway que converta HTTP REST em gRPC
**Para que** o frontend possa consumir APIs REST enquanto serviços usam gRPC internamente

#### Critérios de Aceite:
- Gateway expõe endpoints HTTP REST
- Conversão automática HTTP->gRPC via grpc-gateway
- Suporte a WebSocket para notificações em tempo real
- Middleware de autenticação JWT
- Middleware de CORS, rate limiting, logging
- Health checks e métricas Prometheus

#### Tasks:
- [ ] **Task 7.2.1:** Setup do projeto API Gateway
  ```
  services/api-gateway/
  ├── cmd/
  │   └── server/
  │       └── main.go
  ├── internal/
  │   ├── gateway/
  │   │   ├── gateway.go          # Configuração grpc-gateway
  │   │   └── websocket.go        # Handler WebSocket
  │   ├── middleware/
  │   │   ├── auth.go
  │   │   ├── cors.go
  │   │   ├── ratelimit.go
  │   │   └── logging.go
  │   ├── clients/
  │   │   ├── user_client.go
  │   │   ├── ride_client.go
  │   │   ├── location_client.go
  │   │   └── auth_client.go
  │   └── config/
  │       └── config.go
  ├── Dockerfile
  └── go.mod
  ```

- [ ] **Task 7.2.2:** Implementar configuração do Gateway
  ```go
  // services/api-gateway/internal/config/config.go
  package config

  import (
    "time"
    "github.com/spf13/viper"
  )

  type Config struct {
    Server   ServerConfig
    Services ServiceURLs
    Auth     AuthConfig
    CORS     CORSConfig
  }

  type ServerConfig struct {
    HTTPPort      int
    GRPCPort      int
    ReadTimeout   time.Duration
    WriteTimeout  time.Duration
    IdleTimeout   time.Duration
  }

  type ServiceURLs struct {
    UserService     string
    RideService     string
    LocationService string
    AuthService     string
  }

  type AuthConfig struct {
    JWTSecret      string
    JWTExpiration  time.Duration
    PublicRoutes   []string
  }

  type CORSConfig struct {
    AllowedOrigins   []string
    AllowedMethods   []string
    AllowedHeaders   []string
    ExposedHeaders   []string
    AllowCredentials bool
    MaxAge           int
  }

  func Load() (*Config, error) {
    viper.SetConfigName("config")
    viper.SetConfigType("yaml")
    viper.AddConfigPath("./config")
    viper.AddConfigPath(".")

    viper.SetDefault("server.http_port", 8080)
    viper.SetDefault("server.grpc_port", 9090)

    if err := viper.ReadInConfig(); err != nil {
      return nil, err
    }

    var config Config
    if err := viper.Unmarshal(&config); err != nil {
      return nil, err
    }

    return &config, nil
  }
  ```

- [ ] **Task 7.2.3:** Implementar Gateway HTTP->gRPC
  ```go
  // services/api-gateway/internal/gateway/gateway.go
  package gateway

  import (
    "context"
    "net/http"

    "github.com/grpc-ecosystem/grpc-gateway/v2/runtime"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    ridesharingv1 "github.com/org/ride-sharing/gen/go/ridesharing/v1"
  )

  type Gateway struct {
    mux    *runtime.ServeMux
    config *config.Config
  }

  func New(cfg *config.Config) (*Gateway, error) {
    // Criar mux com opções customizadas
    mux := runtime.NewServeMux(
      runtime.WithErrorHandler(customErrorHandler),
      runtime.WithMetadata(extractMetadata),
      runtime.WithIncomingHeaderMatcher(customHeaderMatcher),
      runtime.WithOutgoingHeaderMatcher(customOutgoingMatcher),
    )

    gw := &Gateway{
      mux:    mux,
      config: cfg,
    }

    if err := gw.registerServices(context.Background()); err != nil {
      return nil, err
    }

    return gw, nil
  }

  func (g *Gateway) registerServices(ctx context.Context) error {
    opts := []grpc.DialOption{
      grpc.WithTransportCredentials(insecure.NewCredentials()),
    }

    // Registrar User Service
    if err := ridesharingv1.RegisterUserServiceHandlerFromEndpoint(
      ctx, g.mux, g.config.Services.UserService, opts,
    ); err != nil {
      return err
    }

    // Registrar Ride Service
    if err := ridesharingv1.RegisterRideServiceHandlerFromEndpoint(
      ctx, g.mux, g.config.Services.RideService, opts,
    ); err != nil {
      return err
    }

    // Registrar Location Service
    if err := ridesharingv1.RegisterLocationServiceHandlerFromEndpoint(
      ctx, g.mux, g.config.Services.LocationService, opts,
    ); err != nil {
      return err
    }

    // Registrar Auth Service
    if err := ridesharingv1.RegisterAuthServiceHandlerFromEndpoint(
      ctx, g.mux, g.config.Services.AuthService, opts,
    ); err != nil {
      return err
    }

    return nil
  }

  func (g *Gateway) ServeHTTP(w http.ResponseWriter, r *http.Request) {
    g.mux.ServeHTTP(w, r)
  }

  func customErrorHandler(ctx context.Context, mux *runtime.ServeMux, marshaler runtime.Marshaler, w http.ResponseWriter, r *http.Request, err error) {
    // Implementar tratamento de erros customizado
    // Converter status gRPC para HTTP apropriado
  }

  func extractMetadata(ctx context.Context, req *http.Request) metadata.MD {
    // Extrair headers importantes para metadata gRPC
    md := metadata.New(nil)
    if auth := req.Header.Get("Authorization"); auth != "" {
      md.Set("authorization", auth)
    }
    if reqID := req.Header.Get("X-Request-ID"); reqID != "" {
      md.Set("x-request-id", reqID)
    }
    return md
  }

  func customHeaderMatcher(key string) (string, bool) {
    // Permitir passar headers específicos para gRPC
    switch key {
    case "X-Request-ID", "X-User-ID", "X-User-Type":
      return key, true
    }
    return runtime.DefaultHeaderMatcher(key)
  }

  func customOutgoingMatcher(key string) (string, bool) {
    // Headers do gRPC response que devem ir para HTTP response
    switch key {
    case "x-request-id":
      return "X-Request-ID", true
    }
    return runtime.DefaultHeaderMatcher(key)
  }
  ```

- [ ] **Task 7.2.4:** Implementar WebSocket Handler
  ```go
  // services/api-gateway/internal/gateway/websocket.go
  package gateway

  import (
    "context"
    "encoding/json"
    "net/http"
    "sync"

    "github.com/gorilla/websocket"
    ridesharingv1 "github.com/org/ride-sharing/gen/go/ridesharing/v1"
    "google.golang.org/grpc"
  )

  var upgrader = websocket.Upgrader{
    ReadBufferSize:  1024,
    WriteBufferSize: 1024,
    CheckOrigin: func(r *http.Request) bool {
      // Implementar validação de origin
      return true
    },
  }

  type WebSocketHandler struct {
    clients     map[string]*websocket.Conn
    mu          sync.RWMutex
    authClient  ridesharingv1.AuthServiceClient
  }

  func NewWebSocketHandler(authClient ridesharingv1.AuthServiceClient) *WebSocketHandler {
    return &WebSocketHandler{
      clients:    make(map[string]*websocket.Conn),
      authClient: authClient,
    }
  }

  func (h *WebSocketHandler) HandleConnection(w http.ResponseWriter, r *http.Request) {
    // Validar token JWT no query param ou header
    token := r.URL.Query().Get("token")
    if token == "" {
      token = r.Header.Get("Authorization")
    }

    // Validar token com Auth Service
    ctx := r.Context()
    validateResp, err := h.authClient.ValidateToken(ctx, &ridesharingv1.ValidateTokenRequest{
      AccessToken: token,
    })
    if err != nil || !validateResp.Valid {
      http.Error(w, "Unauthorized", http.StatusUnauthorized)
      return
    }

    userID := validateResp.UserId

    // Upgrade para WebSocket
    conn, err := upgrader.Upgrade(w, r, nil)
    if err != nil {
      return
    }

    h.mu.Lock()
    h.clients[userID] = conn
    h.mu.Unlock()

    defer func() {
      h.mu.Lock()
      delete(h.clients, userID)
      h.mu.Unlock()
      conn.Close()
    }()

    // Conectar ao NotificationService via gRPC stream
    // e fazer bridge WebSocket <-> gRPC stream
    h.handleMessages(ctx, conn, userID)
  }

  func (h *WebSocketHandler) handleMessages(ctx context.Context, conn *websocket.Conn, userID string) {
    // Implementar lógica de ponte entre WebSocket e gRPC stream
    // Ler mensagens do WebSocket (ClientMessage)
    // Enviar para gRPC stream
    // Receber do gRPC stream (ServerMessage)
    // Enviar para WebSocket

    for {
      var clientMsg ridesharingv1.ClientMessage
      err := conn.ReadJSON(&clientMsg)
      if err != nil {
        break
      }

      // Processar mensagem do cliente
      // Enviar via gRPC para NotificationService
    }
  }

  func (h *WebSocketHandler) BroadcastToUser(userID string, message *ridesharingv1.ServerMessage) error {
    h.mu.RLock()
    conn, ok := h.clients[userID]
    h.mu.RUnlock()

    if !ok {
      return fmt.Errorf("user not connected: %s", userID)
    }

    return conn.WriteJSON(message)
  }
  ```

- [ ] **Task 7.2.5:** Implementar Middleware de Autenticação
  ```go
  // services/api-gateway/internal/middleware/auth.go
  package middleware

  import (
    "context"
    "net/http"
    "strings"

    ridesharingv1 "github.com/org/ride-sharing/gen/go/ridesharing/v1"
  )

  type AuthMiddleware struct {
    authClient   ridesharingv1.AuthServiceClient
    publicRoutes map[string]bool
  }

  func NewAuthMiddleware(client ridesharingv1.AuthServiceClient, publicRoutes []string) *AuthMiddleware {
    routeMap := make(map[string]bool)
    for _, route := range publicRoutes {
      routeMap[route] = true
    }
    return &AuthMiddleware{
      authClient:   client,
      publicRoutes: routeMap,
    }
  }

  func (m *AuthMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      // Verificar se rota é pública
      if m.publicRoutes[r.URL.Path] {
        next.ServeHTTP(w, r)
        return
      }

      // Extrair token do header Authorization
      authHeader := r.Header.Get("Authorization")
      if authHeader == "" {
        http.Error(w, "Missing authorization header", http.StatusUnauthorized)
        return
      }

      token := strings.TrimPrefix(authHeader, "Bearer ")

      // Validar token via Auth Service
      ctx := r.Context()
      validateResp, err := m.authClient.ValidateToken(ctx, &ridesharingv1.ValidateTokenRequest{
        AccessToken: token,
      })

      if err != nil || !validateResp.Valid {
        http.Error(w, "Invalid token", http.StatusUnauthorized)
        return
      }

      // Adicionar user info no contexto
      ctx = context.WithValue(ctx, "user_id", validateResp.UserId)
      ctx = context.WithValue(ctx, "user_type", validateResp.UserType)

      // Adicionar headers customizados para gRPC services
      r.Header.Set("X-User-ID", validateResp.UserId)
      r.Header.Set("X-User-Type", validateResp.UserType.String())

      next.ServeHTTP(w, r.WithContext(ctx))
    })
  }
  ```

- [ ] **Task 7.2.6:** Implementar Middleware de CORS
  ```go
  // services/api-gateway/internal/middleware/cors.go
  package middleware

  import (
    "net/http"
    "strings"
  )

  type CORSMiddleware struct {
    allowedOrigins   []string
    allowedMethods   []string
    allowedHeaders   []string
    exposedHeaders   []string
    allowCredentials bool
    maxAge           int
  }

  func NewCORSMiddleware(cfg *config.CORSConfig) *CORSMiddleware {
    return &CORSMiddleware{
      allowedOrigins:   cfg.AllowedOrigins,
      allowedMethods:   cfg.AllowedMethods,
      allowedHeaders:   cfg.AllowedHeaders,
      exposedHeaders:   cfg.ExposedHeaders,
      allowCredentials: cfg.AllowCredentials,
      maxAge:           cfg.MaxAge,
    }
  }

  func (m *CORSMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      origin := r.Header.Get("Origin")

      if origin != "" && m.isAllowedOrigin(origin) {
        w.Header().Set("Access-Control-Allow-Origin", origin)
      }

      w.Header().Set("Access-Control-Allow-Methods", strings.Join(m.allowedMethods, ", "))
      w.Header().Set("Access-Control-Allow-Headers", strings.Join(m.allowedHeaders, ", "))
      w.Header().Set("Access-Control-Expose-Headers", strings.Join(m.exposedHeaders, ", "))

      if m.allowCredentials {
        w.Header().Set("Access-Control-Allow-Credentials", "true")
      }

      if m.maxAge > 0 {
        w.Header().Set("Access-Control-Max-Age", fmt.Sprintf("%d", m.maxAge))
      }

      if r.Method == http.MethodOptions {
        w.WriteHeader(http.StatusNoContent)
        return
      }

      next.ServeHTTP(w, r)
    })
  }

  func (m *CORSMiddleware) isAllowedOrigin(origin string) bool {
    for _, allowed := range m.allowedOrigins {
      if allowed == "*" || allowed == origin {
        return true
      }
    }
    return false
  }
  ```

- [ ] **Task 7.2.7:** Implementar Middleware de Rate Limiting
  ```go
  // services/api-gateway/internal/middleware/ratelimit.go
  package middleware

  import (
    "net/http"
    "sync"
    "time"

    "golang.org/x/time/rate"
  )

  type RateLimiter struct {
    limiters map[string]*rate.Limiter
    mu       sync.RWMutex
    rate     rate.Limit
    burst    int
  }

  func NewRateLimiter(requestsPerMinute int) *RateLimiter {
    return &RateLimiter{
      limiters: make(map[string]*rate.Limiter),
      rate:     rate.Limit(requestsPerMinute) / 60,
      burst:    requestsPerMinute,
    }
  }

  func (rl *RateLimiter) getLimiter(key string) *rate.Limiter {
    rl.mu.Lock()
    defer rl.mu.Unlock()

    limiter, exists := rl.limiters[key]
    if !exists {
      limiter = rate.NewLimiter(rl.rate, rl.burst)
      rl.limiters[key] = limiter
    }

    return limiter
  }

  func (rl *RateLimiter) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      // Rate limit por user_id se autenticado, senão por IP
      key := r.Header.Get("X-User-ID")
      if key == "" {
        key = r.RemoteAddr
      }

      limiter := rl.getLimiter(key)
      if !limiter.Allow() {
        http.Error(w, "Rate limit exceeded", http.StatusTooManyRequests)
        return
      }

      next.ServeHTTP(w, r)
    })
  }
  ```

- [ ] **Task 7.2.8:** Implementar Middleware de Logging
  ```go
  // services/api-gateway/internal/middleware/logging.go
  package middleware

  import (
    "net/http"
    "time"

    "go.uber.org/zap"
    "github.com/google/uuid"
  )

  type LoggingMiddleware struct {
    logger *zap.Logger
  }

  func NewLoggingMiddleware(logger *zap.Logger) *LoggingMiddleware {
    return &LoggingMiddleware{logger: logger}
  }

  type responseWriter struct {
    http.ResponseWriter
    statusCode int
    bytes      int
  }

  func (rw *responseWriter) WriteHeader(code int) {
    rw.statusCode = code
    rw.ResponseWriter.WriteHeader(code)
  }

  func (rw *responseWriter) Write(b []byte) (int, error) {
    n, err := rw.ResponseWriter.Write(b)
    rw.bytes += n
    return n, err
  }

  func (m *LoggingMiddleware) Handler(next http.Handler) http.Handler {
    return http.HandlerFunc(func(w http.ResponseWriter, r *http.Request) {
      start := time.Now()

      // Gerar ou extrair Request ID
      requestID := r.Header.Get("X-Request-ID")
      if requestID == "" {
        requestID = uuid.New().String()
        r.Header.Set("X-Request-ID", requestID)
      }

      // Wrapper para capturar status code e bytes
      rw := &responseWriter{
        ResponseWriter: w,
        statusCode:     http.StatusOK,
      }

      // Adicionar Request-ID ao response header
      rw.Header().Set("X-Request-ID", requestID)

      next.ServeHTTP(rw, r)

      duration := time.Since(start)

      m.logger.Info("http_request",
        zap.String("request_id", requestID),
        zap.String("method", r.Method),
        zap.String("path", r.URL.Path),
        zap.String("query", r.URL.RawQuery),
        zap.Int("status_code", rw.statusCode),
        zap.Int("response_bytes", rw.bytes),
        zap.Duration("duration", duration),
        zap.String("user_agent", r.UserAgent()),
        zap.String("remote_addr", r.RemoteAddr),
      )
    })
  }
  ```

- [ ] **Task 7.2.9:** Implementar servidor principal
  ```go
  // services/api-gateway/cmd/server/main.go
  package main

  import (
    "context"
    "fmt"
    "net/http"
    "os"
    "os/signal"
    "syscall"
    "time"

    "go.uber.org/zap"
    "google.golang.org/grpc"
    "google.golang.org/grpc/credentials/insecure"

    "github.com/org/ride-sharing/services/api-gateway/internal/config"
    "github.com/org/ride-sharing/services/api-gateway/internal/gateway"
    "github.com/org/ride-sharing/services/api-gateway/internal/middleware"
    ridesharingv1 "github.com/org/ride-sharing/gen/go/ridesharing/v1"
  )

  func main() {
    logger, _ := zap.NewProduction()
    defer logger.Sync()

    cfg, err := config.Load()
    if err != nil {
      logger.Fatal("Failed to load config", zap.Error(err))
    }

    // Criar cliente Auth Service para middleware
    authConn, err := grpc.Dial(
      cfg.Services.AuthService,
      grpc.WithTransportCredentials(insecure.NewCredentials()),
    )
    if err != nil {
      logger.Fatal("Failed to connect to auth service", zap.Error(err))
    }
    defer authConn.Close()

    authClient := ridesharingv1.NewAuthServiceClient(authConn)

    // Criar Gateway
    gw, err := gateway.New(cfg)
    if err != nil {
      logger.Fatal("Failed to create gateway", zap.Error(err))
    }

    // Criar WebSocket Handler
    wsHandler := gateway.NewWebSocketHandler(authClient)

    // Configurar middlewares
    authMW := middleware.NewAuthMiddleware(authClient, cfg.Auth.PublicRoutes)
    corsMW := middleware.NewCORSMiddleware(&cfg.CORS)
    rateLimitMW := middleware.NewRateLimiter(60) // 60 req/min
    loggingMW := middleware.NewLoggingMiddleware(logger)

    // Criar mux principal
    mux := http.NewServeMux()

    // WebSocket endpoint
    mux.HandleFunc("/ws", wsHandler.HandleConnection)

    // Health check
    mux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
      w.WriteHeader(http.StatusOK)
      w.Write([]byte("OK"))
    })

    // API Gateway (com middlewares)
    apiHandler := loggingMW.Handler(
      corsMW.Handler(
        rateLimitMW.Handler(
          authMW.Handler(gw),
        ),
      ),
    )
    mux.Handle("/", apiHandler)

    // Configurar servidor HTTP
    srv := &http.Server{
      Addr:         fmt.Sprintf(":%d", cfg.Server.HTTPPort),
      Handler:      mux,
      ReadTimeout:  cfg.Server.ReadTimeout,
      WriteTimeout: cfg.Server.WriteTimeout,
      IdleTimeout:  cfg.Server.IdleTimeout,
    }

    // Iniciar servidor em goroutine
    go func() {
      logger.Info("Starting API Gateway", zap.Int("port", cfg.Server.HTTPPort))
      if err := srv.ListenAndServe(); err != nil && err != http.ErrServerClosed {
        logger.Fatal("Failed to start server", zap.Error(err))
      }
    }()

    // Graceful shutdown
    quit := make(chan os.Signal, 1)
    signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
    <-quit

    logger.Info("Shutting down server...")
    ctx, cancel := context.WithTimeout(context.Background(), 30*time.Second)
    defer cancel()

    if err := srv.Shutdown(ctx); err != nil {
      logger.Fatal("Server forced to shutdown", zap.Error(err))
    }

    logger.Info("Server exited")
  }
  ```

- [ ] **Task 7.2.10:** Criar arquivo de configuração
  ```yaml
  # services/api-gateway/config/config.yaml
  server:
    http_port: 8080
    grpc_port: 9090
    read_timeout: 30s
    write_timeout: 30s
    idle_timeout: 120s

  services:
    user_service: "user-service.ride-sharing.svc.cluster.local:9090"
    ride_service: "ride-service.ride-sharing.svc.cluster.local:9090"
    location_service: "location-service.ride-sharing.svc.cluster.local:9090"
    auth_service: "auth-service.ride-sharing.svc.cluster.local:9090"

  auth:
    jwt_secret: "change-me-in-production"
    jwt_expiration: 24h
    public_routes:
      - "/v1/auth/login"
      - "/v1/auth/refresh"
      - "/health"

  cors:
    allowed_origins:
      - "http://localhost:3000"
      - "http://localhost:5173"
    allowed_methods:
      - "GET"
      - "POST"
      - "PATCH"
      - "DELETE"
      - "OPTIONS"
    allowed_headers:
      - "Authorization"
      - "Content-Type"
      - "X-Request-ID"
    exposed_headers:
      - "X-Request-ID"
    allow_credentials: true
    max_age: 3600
  ```

- [ ] **Task 7.2.11:** Criar Kubernetes manifests para API Gateway
  ```yaml
  # k8s/services/api-gateway/deployment.yaml
  apiVersion: apps/v1
  kind: Deployment
  metadata:
    name: api-gateway
    namespace: ride-sharing
  spec:
    replicas: 2
    selector:
      matchLabels:
        app: api-gateway
    template:
      metadata:
        labels:
          app: api-gateway
      spec:
        containers:
        - name: api-gateway
          image: api-gateway:latest
          ports:
          - containerPort: 8080
            name: http
          - containerPort: 9090
            name: grpc
          env:
          - name: CONFIG_PATH
            value: "/config/config.yaml"
          volumeMounts:
          - name: config
            mountPath: /config
          livenessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 10
            periodSeconds: 10
          readinessProbe:
            httpGet:
              path: /health
              port: 8080
            initialDelaySeconds: 5
            periodSeconds: 5
        volumes:
        - name: config
          configMap:
            name: api-gateway-config
  ---
  apiVersion: v1
  kind: Service
  metadata:
    name: api-gateway
    namespace: ride-sharing
  spec:
    type: NodePort
    selector:
      app: api-gateway
    ports:
    - name: http
      port: 80
      targetPort: 8080
      nodePort: 30000
    - name: grpc
      port: 9090
      targetPort: 9090
  ---
  apiVersion: v1
  kind: ConfigMap
  metadata:
    name: api-gateway-config
    namespace: ride-sharing
  data:
    config.yaml: |
      # Config content here
  ```

- [ ] **Task 7.2.12:** Atualizar Tiltfile para incluir API Gateway
  ```python
  # Tiltfile (adicionar)

  # API Gateway
  docker_build(
    'api-gateway',
    'services/api-gateway',
    dockerfile='services/api-gateway/Dockerfile',
    live_update=[
      sync('services/api-gateway', '/app'),
      run('go build -o /app/bin/api-gateway ./cmd/server'),
      restart_container(),
    ],
  )
  k8s_yaml('k8s/services/api-gateway/deployment.yaml')
  k8s_resource('api-gateway',
    port_forwards=['8080:8080', '9090:9090'],
    labels=['gateway']
  )

  # Gerar protos automaticamente
  local_resource(
    'proto-gen',
    'make proto-gen',
    deps=['proto'],
    labels=['proto']
  )
  ```

---

## 8. Integrações e Testes E2E

### US-027: Teste E2E do Fluxo de Matching Completo
**Como** QA Engineer  
**Quero** validar o fluxo completo de uma corrida  
**Para que** eu tenha confiança que o sistema funciona end-to-end

#### Critérios de Aceite:
- Teste automatizado simula: rider solicita → motorista aceita → corrida finaliza
- Validação de estados no banco de dados
- Validação de eventos publicados no RabbitMQ
- Tempo total de execução < 60 segundos

#### Tasks:
- [ ] **Task 8.1.1:** Setup de ambiente de testes E2E
  - Docker Compose para teste: todos os serviços + testcontainers
  - Seed data: 1 rider, 3 drivers em posições conhecidas
  
- [ ] **Task 8.1.2:** Implementar teste de happy path
  ```go
  func TestE2E_RideMatching_HappyPath(t *testing.T) {
    // 1. Rider solicita estimativa de tarifa
    fare := requestFare(pickup, destination)
    assert.NotNil(t, fare.FareID)
    
    // 2. Rider cria corrida
    ride := createRide(riderID, fare.FareID, pickup, destination)
    assert.Equal(t, "REQUESTED", ride.Status)
    
    // 3. Aguardar matching (consumir evento via WebSocket)
    offer := waitForDriverOffer(driverID, 30*time.Second)
    assert.Equal(t, ride.RideID, offer.RideID)
    
    // 4. Driver aceita corrida
    acceptRide(driverID, ride.RideID)
    
    // 5. Validar estado final
    rideUpdated := getRide(ride.RideID)
    assert.Equal(t, "ACCEPTED", rideUpdated.Status)
    assert.Equal(t, driverID, rideUpdated.DriverID)
    
    // 6. Validar lock foi liberado
    status := getDriverStatus(driverID)
    assert.False(t, status.IsLocked)
  }
  ```
  
- [ ] **Task 8.1.3:** Implementar teste de rejeição e retry
  ```go
  func TestE2E_RideMatching_DriverRejects(t *testing.T) {
    ride := createRide(riderID, fareID, pickup, destination)
    
    // Driver 1 recebe oferta
    offer1 := waitForDriverOffer(driver1ID, 30*time.Second)
    
    // Driver 1 rejeita
    rejectRide(driver1ID, ride.RideID, "too far")
    
    // Driver 2 recebe oferta
    offer2 := waitForDriverOffer(driver2ID, 30*time.Second)
    assert.Equal(t, ride.RideID, offer2.RideID)
    
    // Driver 2 aceita
    acceptRide(driver2ID, ride.RideID)
    
    // Validar matching final
    rideUpdated := getRide(ride.RideID)
    assert.Equal(t, driver2ID, rideUpdated.DriverID)
  }
  ```
  
- [ ] **Task 8.1.4:** Implementar teste de timeout
  ```go
  func TestE2E_RideMatching_DriverTimeout(t *testing.T) {
    ride := createRide(riderID, fareID, pickup, destination)
    
    offer1 := waitForDriverOffer(driver1ID, 30*time.Second)
    
    // Não responder - aguardar timeout de 20s
    time.Sleep(25 * time.Second)
    
    // Próximo driver deve receber oferta
    offer2 := waitForDriverOffer(driver2ID, 10*time.Second)
    assert.NotNil(t, offer2)
  }
  ```
  
- [ ] **Task 8.1.5:** Implementar teste de corrida completa
  ```go
  func TestE2E_RideLifecycle_Complete(t *testing.T) {
    // 1. Create ride
    // 2. Match com driver
    // 3. Driver inicia corrida
    updateRideStatus(ride.RideID, "IN_PROGRESS")
    
    // 4. Driver envia pings de GPS
    sendLocationPings(driverID, routeCoordinates)
    
    // 5. Rider recebe atualizações de localização
    updates := collectLocationUpdates(riderID, 10*time.Second)
    assert.GreaterOrEqual(t, len(updates), 3)
    
    // 6. Driver completa corrida
    updateRideStatus(ride.RideID, "COMPLETED")
    
    // 7. Validar estado final
    rideUpdated := getRide(ride.RideID)
    assert.Equal(t, "COMPLETED", rideUpdated.Status)
    assert.NotNil(t, rideUpdated.CompletedAt)
  }
  ```

---

### US-028: Testes de Carga e Performance
**Como** SRE  
**Quero** validar que o sistema suporta a carga esperada  
**Para que** eu saiba se a arquitetura escalará em produção

#### Critérios de Aceite:
- Matching de 1000 corridas simultâneas em < 1 minuto
- Ingestão de 10k pings de GPS/segundo
- Latência p99 de APIs < 200ms
- Zero double-booking de motoristas

#### Tasks:
- [ ] **Task 8.2.1:** Setup de ferramentas de teste de carga
  - Instalar k6 ou Locust
  - Scripts de teste para cada endpoint
  
- [ ] **Task 8.2.2:** Teste de carga: Criação de corridas
  ```javascript
  // k6 script
  import http from 'k6/http';
  
  export let options = {
    vus: 100, // 100 virtual users
    duration: '1m',
  };
  
  export default function() {
    let payload = JSON.stringify({
      rider_id: __VU, // ID único por VU
      fare_id: 'valid-fare-id',
      pickup_lat: -23.55,
      pickup_lng: -46.63,
      destination_lat: -23.56,
      destination_lng: -46.64,
    });
    
    http.post('http://localhost:8000/v1/rides', payload, {
      headers: { 'Content-Type': 'application/json' },
    });
  }
  ```
  
- [ ] **Task 8.2.3:** Teste de carga: Ingestão de GPS
  ```go
  func BenchmarkLocationIngestion(b *testing.B) {
    ws := connectWebSocket("wss://localhost:8080/realtime")
    
    b.ResetTimer()
    for i := 0; i < b.N; i++ {
      ping := LocationPing{
        DriverID:  fmt.Sprintf("driver-%d", i%1000),
        Latitude:  -23.55 + rand.Float64()*0.1,
        Longitude: -46.63 + rand.Float64()*0.1,
        Timestamp: time.Now(),
      }
      ws.WriteJSON(ping)
    }
  }
  ```
  
- [ ] **Task 8.2.4:** Validar ausência de double-booking
  ```sql
  -- Query de validação
  SELECT driver_id, COUNT(*) as concurrent_rides
  FROM rides
  WHERE status IN ('ACCEPTED', 'IN_PROGRESS')
    AND driver_id IS NOT NULL
  GROUP BY driver_id
  HAVING COUNT(*) > 1;
  
  -- Resultado esperado: 0 linhas
  ```
  
- [ ] **Task 8.2.5:** Coletar métricas de latência
  - Instrumentar endpoints com Prometheus
  - Criar dashboard Grafana com histogramas de latência
  - Validar p50, p95, p99 em carga

---

## 9. Observabilidade e Monitoramento

### US-029: Implementar Logging Estruturado
**Como** desenvolvedor  
**Quero** logs estruturados em JSON  
**Para que** eu possa fazer queries e análises eficientes

#### Critérios de Aceite:
- Todos os serviços logam em formato JSON
- Campos obrigatórios: timestamp, level, service, trace_id, message
- Logs de erro incluem stack trace
- Logs agregados em stdout (capturados pelo Docker)

#### Tasks:
- [ ] **Task 9.1.1:** Configurar biblioteca de logging (zap ou logrus)
  ```go
  logger := zap.NewProduction()
  logger.Info("ride created",
    zap.String("ride_id", rideID),
    zap.String("rider_id", riderID),
    zap.String("status", "REQUESTED"),
  )
  ```
  
- [ ] **Task 9.1.2:** Implementar middleware de logging HTTP
  - Logar: método, path, status_code, latência, request_id
  - Adicionar trace_id em headers: X-Request-ID
  
- [ ] **Task 9.1.3:** Implementar correlation ID entre serviços
  - Propagar trace_id em chamadas HTTP via header
  - Propagar trace_id em eventos RabbitMQ via campo correlation_id
  
- [ ] **Task 9.1.4:** Configurar níveis de log por ambiente
  - Dev: DEBUG
  - Staging: INFO
  - Prod: WARN + ERROR

---

### US-030: Implementar Métricas com Prometheus
**Como** SRE  
**Quero** coletar métricas de performance e negócio  
**Para que** eu possa monitorar a saúde do sistema

#### Critérios de Aceite:
- Endpoint `/metrics` exposto em cada serviço
- Métricas RED (Rate, Errors, Duration) para APIs
- Métricas de negócio: rides_created, matches_found, etc
- Prometheus scraping configurado

#### Tasks:
- [ ] **Task 9.2.1:** Adicionar Prometheus ao docker-compose.yml
  ```yaml
  prometheus:
    image: prom/prometheus:latest
    volumes:
      - ./prometheus/prometheus.yml:/etc/prometheus/prometheus.yml
    ports:
      - "9090:9090"
  ```
  
- [ ] **Task 9.2.2:** Configurar scrape targets
  ```yaml
  # prometheus/prometheus.yml
  scrape_configs:
    - job_name: 'user-service'
      static_configs:
        - targets: ['user-service:9100']
    - job_name: 'ride-service'
      static_configs:
        - targets: ['ride-service:9100']
    # ... outros serviços
  ```
  
- [ ] **Task 9.2.3:** Instrumentar aplicações Go
  ```go
  import "github.com/prometheus/client_golang/prometheus/promhttp"
  
  // Expor endpoint /metrics
  http.Handle("/metrics", promhttp.Handler())
  go http.ListenAndServe(":9100", nil)
  ```
  
- [ ] **Task 9.2.4:** Criar métricas customizadas
  ```go
  var (
    ridesCreated = prometheus.NewCounter(prometheus.CounterOpts{
      Name: "rides_created_total",
      Help: "Total number of rides created",
    })
    
    matchingDuration = prometheus.NewHistogram(prometheus.HistogramOpts{
      Name:    "matching_duration_seconds",
      Help:    "Time to find a driver match",
      Buckets: prometheus.ExponentialBuckets(0.1, 2, 10),
    })
    
    driversOnline = prometheus.NewGauge(prometheus.GaugeOpts{
      Name: "drivers_online",
      Help: "Number of drivers currently online",
    })
  )
  
  func init() {
    prometheus.MustRegister(ridesCreated, matchingDuration, driversOnline)
  }
  ```

---

### US-031: Criar Dashboards Grafana
**Como** operador  
**Quero** visualizar métricas em dashboards  
**Para que** eu possa monitorar o sistema de forma visual

#### Critérios de Aceite:
- Dashboard de overview: requests/s, latência, errors
- Dashboard de matching: tempo médio, taxa de sucesso, tentativas
- Dashboard de location: pings/s, motoristas online, locks ativos
- Dashboards provisionados automaticamente

#### Tasks:
- [ ] **Task 9.3.1:** Adicionar Grafana ao docker-compose.yml
  ```yaml
  grafana:
    image: grafana/grafana:latest
    environment:
      GF_SECURITY_ADMIN_PASSWORD: admin
    ports:
      - "3000:3000"
    volumes:
      - ./grafana/provisioning:/etc/grafana/provisioning
  ```
  
- [ ] **Task 9.3.2:** Configurar datasource Prometheus
  ```yaml
  # grafana/provisioning/datasources/prometheus.yml
  apiVersion: 1
  datasources:
    - name: Prometheus
      type: prometheus
      url: http://prometheus:9090
      isDefault: true
  ```
  
- [ ] **Task 9.3.3:** Criar dashboard de sistema (overview.json)
  - Panel: Request Rate (qps)
  - Panel: Latência p50, p95, p99
  - Panel: Error Rate (%)
  - Panel: CPU e Memória por serviço
  
- [ ] **Task 9.3.4:** Criar dashboard de matching (matching.json)
  - Panel: Matching Duration (histogram)
  - Panel: Match Success Rate
  - Panel: Tentativas por Match (avg)
  - Panel: Timeouts e Rejeições
  
- [ ] **Task 9.3.5:** Criar dashboard de location (location.json)
  - Panel: GPS Pings/s
  - Panel: Motoristas Online (gauge)
  - Panel: Locks Ativos (gauge)
  - Panel: Latência de GEORADIUS

---

### US-032: Configurar Alertas Básicos
**Como** SRE  
**Quero** receber alertas quando métricas críticas ultrapassarem limites  
**Para que** eu possa reagir rapidamente a incidentes

#### Critérios de Aceite:
- Alertas via Prometheus Alertmanager
- Notificações via Slack (ou webhook)
- Alertas: alta latência, alta taxa de erro, serviço down

#### Tasks:
- [ ] **Task 9.4.1:** Adicionar Alertmanager ao docker-compose.yml
  ```yaml
  alertmanager:
    image: prom/alertmanager:latest
    ports:
      - "9093:9093"
    volumes:
      - ./alertmanager/config.yml:/etc/alertmanager/config.yml
  ```
  
- [ ] **Task 9.4.2:** Configurar rotas de alerta
  ```yaml
  # alertmanager/config.yml
  route:
    receiver: 'slack'
    group_by: ['alertname', 'service']
    group_wait: 10s
    group_interval: 5m
    repeat_interval: 3h
  
  receivers:
    - name: 'slack'
      slack_configs:
        - api_url: 'https://hooks.slack.com/services/YOUR/WEBHOOK/URL'
          channel: '#alerts'
  ```
  
- [ ] **Task 9.4.3:** Criar regras de alerta
  ```yaml
  # prometheus/alerts.yml
  groups:
    - name: api_alerts
      rules:
        - alert: HighLatency
          expr: histogram_quantile(0.99, http_request_duration_seconds_bucket) > 1
          for: 5m
          labels:
            severity: warning
          annotations:
            summary: "High latency on {{ $labels.service }}"
        
        - alert: HighErrorRate
          expr: rate(http_requests_total{status=~"5.."}[5m]) > 0.05
          for: 2m
          labels:
            severity: critical
          annotations:
            summary: "High error rate on {{ $labels.service }}"
        
        - alert: ServiceDown
          expr: up == 0
          for: 1m
          labels:
            severity: critical
          annotations:
            summary: "Service {{ $labels.job }} is down"
  ```

---

## Resumo de Entregáveis

### Por Serviço:
- **User Service:** CRUD de usuários, health checks
- **Location Service:** WebSocket GPS, geo-index Redis, distributed locking, API de proximidade
- **Ride Service:** Estimativa de tarifa, CRUD de corridas, máquina de estados, integração RabbitMQ
- **Matching Service:** Consumer de eventos, algoritmo de matching, Temporal workflows, retry logic
- **Notification Service:** WebSocket hub, consumers de eventos, roteamento de mensagens, tracking real-time
- **API Gateway:** Kong configurado, rotas, plugins (CORS, rate-limit, JWT)

### Infraestrutura:
- Docker Compose completo (PostgreSQL, Redis, RabbitMQ, Temporal, Kong, Prometheus, Grafana)
- Migrations versionadas
- Scripts de provisionamento
- Configurações de ambiente

### Qualidade:
- Testes unitários (>80% cobertura)
- Testes E2E de fluxos críticos
- Testes de carga
- Logging estruturado
- Métricas Prometheus
- Dashboards Grafana
- Alertas configurados

---

## Priorização Sugerida

### Sprint 1 (Fundação):
- US-001, US-002, US-003: Infraestrutura
- US-004: User Service básico
- US-006, US-007: Location Service (ingestão + busca)

### Sprint 2 (Core Matching):
- US-008: Distributed Locking
- US-010, US-011: Ride Service (tarifa + criação)
- US-016, US-017, US-018: Matching Service básico

### Sprint 3 (Fluxo Completo):
- US-012, US-013, US-014, US-015: Status updates e aceite/rejeição
- US-021, US-022, US-023: Notification Service
- US-027: Testes E2E

### Sprint 4 (Resiliência):
- US-019: Timeouts e Temporal
- US-009: Gestão de status de motoristas
- US-024: Tracking real-time

### Sprint 5 (Gateway e Observabilidade):
- US-025, US-026: API Gateway + Auth
- US-029, US-030, US-031, US-032: Observabilidade completa
- US-028: Testes de carga

---

## Notas Finais

Este documento serve como roadmap de implementação do sistema de ride-sharing descrito nas RFCs 0001 e 0002. As User Stories foram desenhadas para serem incrementais e testáveis, permitindo entregas contínuas de valor.

**Pontos de Atenção:**
- Priorizar strong consistency no matching (locks no Redis)
- Monitorar latência p99 de todas as APIs críticas
- Testar cenários de falha (Redis down, RabbitMQ down, etc.)
- Documentar decisões arquiteturais conforme implementação avança
