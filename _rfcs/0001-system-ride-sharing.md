# RFC: Sistema de Ride-Sharing
**Data:** 21 de Fevereiro de 2026
**Versão:** 1.1.0

---

## 1. Visão Geral e Escopo

Este documento propõe a arquitetura de uma plataforma de mobilidade urbana focada na conexão em tempo real entre passageiros e motoristas. O objetivo é construir um motor de pareamento (matching) de ultra-baixa latência e alta consistência.

### Core Features (MVP)

* **Passageiros:** Estimativa de tarifa, solicitação, rastreamento e cancelamento pré-início.
* **Motoristas:** Gestão de disponibilidade, aceite/recusa de ofertas e telemetria em tempo real.
* **Matching:** Algoritmo de busca por proximidade com garantia de exclusividade de oferta.

### Fora de Escopo

* Sistemas de avaliação (Ratings)
* Agendamento prévio (Scheduled Rides)
* Múltiplas categorias de veículos

---

## 2. Requisitos Não-Funcionais

* **Disponibilidade:** 99.99% (Critical Path).
* **Latência de Matching:** < 1 minuto para fechar o ciclo de pareamento.
* **Consistência:** **Strong Consistency** no matching para evitar "double-booking" de motoristas.
* **Escalabilidade:** Ingestão de ~2 milhões de pings de localização por segundo.
* **Resiliência:** Persistência de estado via Event Sourcing para recuperação de desastres.
* **Isolamento de Dados:** Padrão **Database-per-Service** para garantir autonomia e desacoplamento entre microsserviços.

---

## 3. Arquitetura de Alto Nível

A arquitetura adota um modelo **Event-Driven** sobre microsserviços, utilizando o **RabbitMQ** como message broker para comunicação assíncrona com roteamento flexível através de exchanges e routing keys.

### Padrão Database-per-Service

Cada microsserviço possui sua **própria instância de banco de dados PostgreSQL**, garantindo:

* **Autonomia:** Serviços não dependem do schema de outros serviços
* **Escalabilidade Independente:** Cada banco pode ser escalado conforme a demanda específica do serviço
* **Resiliência:** Falha no banco de um serviço não afeta os demais
* **Evolução Independente:** Mudanças de schema não impactam outros serviços
* **Isolamento de Dados:** Nenhum acesso direto cross-service ao banco de dados

**Instâncias de Banco de Dados:**
* **user-service-db:** Gerencia dados de passageiros e motoristas
* **ride-service-db:** Gerencia ciclo de vida das corridas e tarifas
* **location-service-db:** (Opcional) Metadados de configuração - Redis como primary storage
* **matching-service-db:** (Opcional) Logs de matching e métricas

**Comunicação entre Serviços:**
* Dados compartilhados são trocados **exclusivamente via eventos (RabbitMQ)** ou **APIs REST/gRPC**
* Nenhum serviço acessa diretamente o banco de dados de outro serviço

A arquitetura adota um modelo **Event-Driven** sobre microsserviços, utilizando o **RabbitMQ** como message broker para comunicação assíncrona com roteamento flexível através de exchanges e routing keys.

### Componentes Principais

* **API Gateway:** Ponto único de entrada (Kong ou AWS AppSync) para AuthN/AuthZ e Rate Limiting.
* **User Service:** Gerenciador de passageiros e motoristas.
* **Ride Service:** Orquestrador do ciclo de vida da corrida e cálculo de tarifas.
* **Location Service:** Microsserviço independente responsável por indexação geoespacial, gerenciamento de posições de motoristas e distributed locking. Expõe API REST/gRPC para consultas de proximidade e controle de disponibilidade. Utiliza Redis como storage layer para operações de alta performance (GEORADIUS, locks).
* **Matching Service:** Cérebro do sistema; consome solicitações e encontra o melhor par através de chamadas à API do Location Service.
* **Notification/Websocket Service:** Mantém túneis persistentes para atualizações push.

### Comunicação entre Serviços

* **Síncrona (Request/Response):** Matching Service → Location Service (REST/gRPC)
* **Assíncrona (Event-Driven):** RabbitMQ com exchanges do tipo topic para eventos de negócio (ride_requested, match_found, etc.) com roteamento baseado em routing keys
* **Real-time (Bidirectional):** WebSocket para tracking de localização e notificações

---

## 4. Design de API e Comunicação

### Híbrido: REST + WebSockets

| Ação | Protocolo | Endpoint |
| --- | --- | --- |
| Solicitar Tarifa | **REST (POST)** | `/v1/fares` |
| Criar Corrida | **REST (POST)** | `/v1/rides` |
| Aceitar Corrida | **REST (PATCH)** | `/v1/rides/:id/accept` |
| Stream de Localização | **WebSocket** | `wss://api.domain.com/realtime` |

### API do Location Service (Comunicação Interna)

O **Location Service** expõe endpoints REST (ou gRPC para melhor performance) consumidos pelos demais microsserviços:

| Operação | Método | Endpoint | Descrição |
| --- | --- | --- | --- |
| Atualizar Posição | **POST** | `/v1/location/drivers/:id/location` | Recebe ping de GPS e atualiza Redis (GEOADD) |
| Buscar Próximos | **GET** | `/v1/location/drivers/nearby?lat=X&lng=Y&radius=5000` | Retorna lista de motoristas disponíveis via GEORADIUS |
| Adquirir Lock | **POST** | `/v1/location/drivers/:id/lock` | Cria distributed lock com TTL (SET NX EX 15) |
| Liberar Lock | **DELETE** | `/v1/location/drivers/:id/lock` | Remove lock manual ou por timeout |
| Status do Motorista | **GET** | `/v1/location/drivers/:id/status` | Verifica se motorista está locked ou disponível |

**Exemplo de Request (Buscar Próximos):**
```json
GET /v1/location/drivers/nearby?lat=-23.55&lng=-46.63&radius=5000&limit=10

Response:
{
  "drivers": [
    {
      "driver_id": "uuid-driver-1",
      "distance_meters": 1200,
      "last_seen": "2026-02-21T10:05:00Z",
      "status": "available"
    }
  ]
}
```

**Exemplo de Request (Adquirir Lock):**
```json
POST /v1/location/drivers/uuid-driver-1/lock
{
  "ride_id": "uuid-ride-789",
  "ttl_seconds": 15
}

Response:
{
  "locked": true,
  "expires_at": "2026-02-21T10:05:15Z"
}
```

---

## 5. Modelagem de Dados

### Camada Transacional (PostgreSQL)

Utilizada para dados que exigem conformidade ACID e integridade referencial.

* **Tabelas:** `users` (riders/drivers), `rides`, `fares`.
* **RideUpdates (Ledger):** Tabela imutável de log de eventos. Cada mudança de estado (`REQUESTED` -> `MATCHED` -> `PICKED_UP`) gera um registro para auditoria e resolução de conflitos.

### Camada de Velocidade (Redis - Gerenciado pelo Location Service)

O **Location Service** encapsula o acesso ao Redis, expondo operações através de sua API:

* **Geo-Index:** Uso interno de `GEOADD` e `GEORADIUS` para indexar a posição dos motoristas ativos.
* **Distributed Lock:** Gerenciamento de estado de ocupação do motorista via `SET NX EX`.
* **Outros serviços não acessam o Redis diretamente** - toda comunicação passa pela API do Location Service.

## 5.1. Modelagem de Dados (PostgreSQL)

Nesta camada, focamos em **Normalização** e **Auditoria**. Utilizaremos tipos de dados nativos do Postgres (como `UUID` e `JSONB`) para flexibilidade e performance.

### Tabela: `users`

Armazena a identidade base. A distinção entre Rider e Driver é feita por roles ou tabelas de perfil estendidas.
| Coluna | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `UUID` (PK) | Identificador único universal. |
| `full_name` | `VARCHAR(255)` | Nome completo. |
| `email` | `VARCHAR(255)` | Único, indexado para login. |
| `user_type` | `ENUM` | `RIDER`, `DRIVER`. |
| `created_at` | `TIMESTAMPTZ` | Timestamp com fuso horário. |
| `updated_at` | `TIMESTAMPTZ` | Timestamp com fuso horário. |
| `deleted_at` | `TIMESTAMPTZ` | Timestamp com fuso horário. |

### Tabela: `rides` (O Coração do Sistema)

| Coluna | Tipo | Descrição |
| --- | --- | --- |
| `id` | `UUID` (PK) | ID da corrida. |
| `rider_id` | `UUID` (FK) | Referência ao passageiro. |
| `driver_id` | `UUID` (FK, NULL) | Referência ao motorista (nulo até o match). |
| `fare_id` | `UUID` (FK) | Referência à estimativa de preço aceita. |
| `status` | `ENUM` | `REQUESTED`, `MATCHING`, `ACCEPTED`, `IN_PROGRESS`, `COMPLETED`, `CANCELLED`. |
| `pickup_location` | `GEOMETRY(Point, 4326)` | Coordenadas de origem (PostGIS). |
| `destination_location` | `GEOMETRY(Point, 4326)` | Coordenadas de destino (PostGIS).. |

### Tabela: `ride_updates` (Immutable Ledger)

Essencial para resolver disputas e depurar falhas de estado.
| Coluna | Tipo | Descrição |
| :--- | :--- | :--- |
| `id` | `BIGSERIAL` (PK) | ID sequencial para ordenação. |
| `ride_id` | `UUID` (FK) | Indexado para busca rápida por corrida. |
| `old_status` | `ENUM` | Status anterior. |
| `new_status` | `ENUM` | Status atualizado. |
| `changed_at` | `TIMESTAMPTZ` | Momento da transição. |
| `metadata` | `JSONB` | Dados extras (ex: motivo do cancelamento). |

---

## 6. Deep Dives e Soluções Técnicas

### 6.1. Ingestão de Localização (High Throughput)

Para suportar 2M de updates/seg, a arquitetura utiliza o seguinte fluxo:

1. **Client → Location Service:** Apps de motoristas enviam pings de GPS via WebSocket para o **Location Service** (batching local de 3 segundos para reduzir tráfego).
2. **Location Service → Redis:** O serviço processa os pings de forma assíncrona e atualiza o geo-index via `GEOADD`.
3. **Isolamento:** O banco relacional *nunca* recebe pings de telemetria direta, apenas o estado final da corrida (início, fim).

**Arquitetura de Escalabilidade:**
- Múltiplas instâncias do Location Service atrás de Load Balancer.
- Redis Cluster para sharding geográfico (ex: shard por cidade).
- Filas internas (Go channels ou RabbitMQ work queues) para buffering de pings em caso de picos.

**Exemplo de Payload WebSocket:**
```json
{
  "driver_id": "uuid-driver-1",
  "latitude": -23.5505,
  "longitude": -46.6333,
  "timestamp": "2026-02-21T10:05:30Z",
  "heading": 180,
  "speed_kmh": 45
}
```

### 6.2. Concorrência e Locking no Matching

Para evitar que dois passageiros "puxem" o mesmo motorista, o sistema utiliza **Distributed Locking** gerenciado pelo **Location Service**:

**Fluxo de Matching com Lock:**
1. **Matching Service** consome evento `ride_requested` do RabbitMQ via fila dedicada.
2. **Matching Service** chama `GET /internal/drivers/nearby` no **Location Service** para obter lista de motoristas próximos.
3. **Matching Service** itera sobre os candidatos e tenta adquirir lock via `POST /internal/drivers/:id/lock` (TTL de 15 segundos).
4. Se o lock for adquirido com sucesso, o **Matching Service** publica evento `match_candidate_found` em uma exchange do RabbitMQ.
5. Se o motorista recusar ou o TTL expirar, o **Location Service** libera o lock automaticamente (Redis TTL) ou via `DELETE /internal/drivers/:id/lock`.

**Implementação no Redis (dentro do Location Service):**
```redis
SET driver:uuid-123:lock ride:uuid-789 NX EX 15
```

**Vantagens dessa abordagem:**
- Lock atomicity garantida pelo Redis.
- TTL automático previne deadlocks.
- Location Service encapsula toda lógica de lock, facilitando mudanças futuras (ex: migrar para DynamoDB).

### 6.3. Timeouts e Resiliência (Temporal)

Utilizaremos o **Temporal.io** para gerenciar o workflow da corrida. Se um motorista não responder ao evento de `match_offered` dentro de 20 segundos, o workflow dispara automaticamente um evento de `retry_matching`, garantindo que a solicitação não fique "pendurada" no sistema.

---

## 7. Fluxo de Execução (Step-by-Step)

1. **Request:** Passageiro envia `POST /rides`. O **Ride Service** persiste a intenção no Postgres e publica `ride_requested` no RabbitMQ através de uma exchange do tipo topic.

2. **Matching:** O **Matching Service** consome o evento via fila dedicada e executa:
   - Chama `GET /internal/drivers/nearby?lat=-23.55&lng=-46.63&radius=5000` no **Location Service** para obter motoristas próximos.
   - Itera sobre os candidatos ordenados por distância/rating.
   - Tenta adquirir lock via `POST /internal/drivers/:id/lock` no **Location Service**.
   - Se bem-sucedido, publica evento `match_candidate_found` no RabbitMQ com routing key apropriada.

3. **Dispatch:** O **Notification Service** consome `match_candidate_found` de sua fila e envia push/WebSocket para o app do motorista com detalhes da corrida.

4. **Acceptance:**
   - Motorista aceita via `PATCH /rides/:id/accept`.
   - **Ride Service** atualiza status para `MATCHED` no Postgres.
   - **Ride Service** chama `DELETE /internal/drivers/:id/lock` no **Location Service** para liberar o lock.
   - **Ride Service** notifica o passageiro publicando evento no RabbitMQ → Notification Service consome via fila.

5. **Tracking:**
   - Motorista inicia deslocamento; app envia pings de GPS via WebSocket para o **Location Service**.
   - **Location Service** atualiza posição no Redis e publica eventos de movimentação no RabbitMQ.
   - **Notification Service** consome eventos e transmite via WebSocket para o passageiro (visualização em tempo real no mapa).

## 7.1. Estrutura de Mensagens (RabbitMQ)

Para o sistema de mensagens, utilizaremos **JSON** (ou Protobuf em cenários de escala extrema) com cabeçalhos de rastreabilidade (`correlation_id`) e properties do RabbitMQ para roteamento.

### Exchanges e Routing Keys (RabbitMQ)

O sistema utiliza **Topic Exchanges** para roteamento flexível:

* **Exchange:** `rides.events` (tipo: topic)
  * Routing Key: `ride.requested` - Solicitações de corrida
  * Routing Key: `ride.matched` - Corridas com match confirmado
  * Routing Key: `ride.started` - Corridas iniciadas
  * Routing Key: `ride.completed` - Corridas finalizadas
  * Routing Key: `ride.cancelled` - Corridas canceladas

* **Exchange:** `matching.events` (tipo: topic)
  * Routing Key: `matching.driver_offered` - Oferta enviada ao motorista
  * Routing Key: `matching.driver_accepted` - Motorista aceitou
  * Routing Key: `matching.driver_rejected` - Motorista recusou
  * Routing Key: `matching.timeout` - Timeout de resposta

* **Exchange:** `location.events` (tipo: topic)
  * Routing Key: `location.driver.updated` - Atualização de posição
  * Routing Key: `location.driver.online` - Motorista ficou online
  * Routing Key: `location.driver.offline` - Motorista ficou offline

**Vantagens do Roteamento com Topic Exchange:**
- Consumidores podem fazer bind com padrões (ex: `ride.*`, `matching.driver.*`)
- Múltiplos consumidores podem processar o mesmo evento com propósitos diferentes
- Baixa latência no processamento de mensagens individuais
- Facilita implementação de dead letter queues e retry policies

### Tópico: `ride.requested`

Publicado pelo *Ride Service* quando o passageiro clica em "Solicitar".

```json
{
  "event_id": "uuid-v4",
  "correlation_id": "trace-123",
  "timestamp": "2026-02-21T00:10:00Z",
  "payload": {
    "ride_id": "ride-789",
    "rider_id": "rider-456",
    "pickup": { "lat": -23.55, "lng": -46.63 },
    "destination": { "lat": -23.56, "lng": -46.64 },
    "estimated_fare": 25.50
  }
}
```

### Tópico: `matching.driver_offered`

Publicado pelo *Matching Service* após adquirir o lock no Redis. O *Notification Service* consome isso para disparar o Push/WebSocket.

```json
{
  "event_id": "uuid-v4",
  "payload": {
    "ride_id": "ride-789",
    "driver_id": "driver-101",
    "expires_at": "2026-02-21T00:10:20Z", // TTL de 20s
    "distance_to_pickup_meters": 1200
  }
}
```

### Tópico: `ride.status_changed`

Mensagem broadcast para atualizar outros sistemas (Billing, Analytics, Fraud).

```json
{
  "event_id": "uuid-v4",
  "payload": {
    "ride_id": "ride-789",
    "driver_id": "driver-101",
    "new_status": "ACCEPTED",
    "event_time": "2026-02-21T00:10:05Z"
  }
}
```
---

## 8. Diagrama de Comunicação entre Serviços

### Arquitetura de Comunicação (Matching Flow)

```
                                    ┌─────────────┐                    
                                    │   Client    │                    
                                    │  (Rider)    │                    
                                    └──────┬──────┘                    
                                           │ POST /rides
                                           │ 
                                           ▼ 
                                   ┌──────────────┐
                                   │ API Gateway  │
                                   └───────┬──────┘
                                           │
                                           │
                                           │ 
                                           ▼ 
                                    ┌──────────────┐
                                    │ Ride Service │
                                    └──────┬───────┘
                                           │
                    ┌──────────────────────┼─────────────────┐
                    │                      │                 │
                    │ Persist              │ Publish         │
                    ▼                      ▼                 │
             ┌─────────────┐        ┌───────────┐            │
             │  PostgreSQL │        │  RabbitMQ │            │
             │   (rides)   │        │ (events)  │            │
             └─────────────┘        └────┬──────┘            │
                                         │                   │
                         ride_requested  │                   │
                                         ▼                   │
                                  ┌───────────────┐          │
                                  │    Matching   │          │
                                  │    Service    │          │
                                  └───────┬───────┘          │
                                          │                  │
                    ┌─────────────────────┼──────────────────┤
                    │ GET /nearby         │ POST /lock       │
                    ▼                     ▼                  │
            ┌────────────────────────────────────┐           │
            │      Location Service              │           │
            │  ┌──────────┐    ┌──────────┐      │           │
            │  │  Redis   │    │  Redis   │      │           │
            │  │ GeoIndex │    │  Locks   │      │           │
            │  └──────────┘    └──────────┘      │           │ 
            └────────────────────────────────────┘           │
                                          │                  │
                         match_found      │                  │
                                          ▼                  │
                                    ┌────────────┐           │
                                    │  RabbitMQ  │           │
                                    └────┬───────┘           │
                                         │                   │
                                         ▼                   │
                                ┌─────────────────┐          │
                                │  Notification   │          │
                                │    Service      │          │
                                └────────┬────────┘          │
                                         │                   │
                                         │ WebSocket Push    │
                                         ▼                   │
                                  ┌──────────┐               │
                                  │  Driver  │               │
                                  │   App    │               │
                                  └──────┬───┘               │
                                         │                   │
                           PATCH /accept │                   │
                                         └───────────────────┘
```

### Responsabilidades de Cada Serviço

**Location Service:**
- Recebe pings de GPS via WebSocket de apps de motoristas
- Mantém geo-index em Redis (GEOADD/GEORADIUS)
- Gerencia distributed locks para matching exclusivo
- Expõe API REST/gRPC para consultas de proximidade
- **NÃO acessa** diretamente o Postgres (stateless para geolocalização)

**Matching Service:**
- Consome eventos do RabbitMQ via fila dedicada (`ride_requested`)
- Chama Location Service via HTTP/gRPC para buscar candidatos
- Aplica lógica de negócio (score, rating, tempo de espera)
- Adquire locks via Location Service API
- Publica evento `match_found` no RabbitMQ com routing key apropriada

**Ride Service:**
- Gerencia estado transacional no Postgres
- Orquestra workflow da corrida
- Libera locks via Location Service quando match é aceito/cancelado
- Publica eventos de mudança de estado no RabbitMQ através das exchanges

**Notification Service:**
- Mantém conexões WebSocket com apps (passageiros e motoristas)
- Consome eventos do RabbitMQ através de múltiplas filas (com binding patterns flexíveis)
- Roteia mensagens para clientes conectados baseado em user_id/driver_id
- Gerencia presença online de usuários

---

## 9. Considerações de Implementação

### Vantagens da Abordagem com Location Service Independente

1. **Encapsulamento:** Redis é um detalhe de implementação - pode ser trocado por PostGIS, DynamoDB, etc.
2. **Escalabilidade:** Location Service pode escalar independentemente do volume de matching.
3. **Reutilização:** Outros serviços (Analytics, Fraud Detection) podem consultar posições via API padronizada.
4. **Testabilidade:** Mock da API do Location Service simplifica testes do Matching Service.
5. **Monitoramento:** Métricas centralizadas de latência de queries geoespaciais.

### Trade-offs

- **Latência Extra:** Hop de rede adicional (Matching → Location Service → Redis) adiciona ~1-3ms.
- **Complexidade Operacional:** Mais um serviço para deployar, monitorar e escalar.
- **Ponto Único de Falha:** Se Location Service cair, o matching para completamente (mitigado por múltiplas réplicas).

**Decisão:** Para um sistema de produção com requisitos de 99.99% uptime e evolução de longo prazo, os benefícios de isolamento superam o overhead de latência.
```
