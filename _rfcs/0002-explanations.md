# RFC 0002: Explicações Técnicas - Sistema de Ride-Sharing
**Data:** 21 de Fevereiro de 2026
**Versão:** 1.0.0

---

## 1. Distributed Locking com Redis

### 1.1. O que é o Lock no Redis?

O **Distributed Lock** (bloqueio distribuído) é um mecanismo de sincronização que garante que apenas **um processo em toda a infraestrutura** possa acessar um recurso crítico por vez, mesmo quando múltiplas instâncias dos serviços estão rodando em paralelo.

No contexto do Redis, implementamos locks utilizando o comando:

```redis
SET driver:uuid-123:lock ride:uuid-789 NX EX 15
```

**Breakdown do comando:**
- `SET`: Define uma chave com valor
- `driver:uuid-123:lock`: Nome da chave (lock para o motorista específico)
- `ride:uuid-789`: Valor armazenado (ID da corrida que "trancou" o motorista)
- `NX`: **N**ot e**X**ists - só cria se a chave NÃO existir (atomicidade)
- `EX 15`: Expira automaticamente após 15 segundos (TTL)

### 1.2. Para que serve no contexto do Sistema de Ride-Sharing?

O lock resolve o problema crítico de **double-booking** de motoristas. Considere o seguinte cenário sem locks:

**Problema (sem lock):**
```
t0: Passageiro A solicita corrida em São Paulo
t1: Passageiro B solicita corrida em São Paulo
t2: Matching Service (instância 1) busca motoristas próximos → encontra Motorista X
t3: Matching Service (instância 2) busca motoristas próximos → encontra Motorista X
t4: Ambos os sistemas enviam oferta para Motorista X simultaneamente
t5: Motorista X aceita corrida do Passageiro A
t6: Sistema atualiza status de AMBAS as corridas como MATCHED
❌ CONFLITO: Motorista X está alocado para 2 corridas ao mesmo tempo!
```

**Solução (com lock):**
```
t0: Passageiro A solicita corrida
t1: Passageiro B solicita corrida
t2: Matching Service (instância 1) busca motoristas → encontra Motorista X
t3: Matching Service (instância 1) tenta adquirir lock:
    POST /v1/location/drivers/uuid-X/lock
    → Redis retorna: {"locked": true, "expires_at": "..."}
t4: Matching Service (instância 2) busca motoristas → encontra Motorista X
t5: Matching Service (instância 2) tenta adquirir lock:
    POST /v1/location/drivers/uuid-X/lock
    → Redis retorna: {"locked": false, "reason": "already_locked"}
t6: Matching Service (instância 1) envia oferta para Motorista X
t7: Matching Service (instância 2) pula Motorista X e busca próximo candidato
✅ SUCESSO: Apenas uma oferta por motorista, sem conflitos
```

### 1.3. Características Importantes

#### Atomicidade
O comando `SET NX` é **atômico** no Redis, ou seja:
- Se duas requisições tentarem criar a mesma chave simultaneamente
- Apenas uma terá sucesso
- A outra receberá resposta negativa instantaneamente

#### TTL Automático (Time To Live)
O lock expira automaticamente após 15 segundos por motivos de segurança:

**Cenário 1: Motorista recusa a oferta**
```
t0: Lock criado (TTL = 15s)
t5: Motorista recusa
t5: Matching Service libera lock manualmente (DELETE)
→ Motorista fica disponível imediatamente para próxima oferta
```

**Cenário 2: Matching Service trava/cai**
```
t0: Lock criado (TTL = 15s)
t3: Matching Service sofre crash antes de liberar o lock
t15: Redis expira o lock automaticamente
→ Sistema se "auto-recupera" sem intervenção manual
```

#### Prevenção de Deadlocks
Sem TTL, poderíamos ter **deadlocks permanentes**:
- Matching Service adquire lock
- Crash acontece antes de liberar
- Motorista fica "preso" indefinidamente como "ocupado"
- Nunca mais recebe ofertas

Com TTL de 15 segundos:
- Tempo suficiente para o motorista responder (limite de 20s no Temporal)
- Curto o bastante para não impactar disponibilidade do motorista
- Balance entre consistência e disponibilidade

### 1.4. Fluxo Completo com Lock

```
┌─────────────────┐
│ Matching Service│
└────────┬────────┘
         │ 1. Consume evento ride_requested
         │
         ▼
┌─────────────────────────────────────────┐
│ GET /v1/location/drivers/nearby         │
│→ Retorna: [Driver A, Driver B, Driver C]│
└────────┬────────────────────────────────┘
         │
         │ 2. Itera sobre candidatos
         ▼
┌──────────────────────────────────────────┐
│ Loop: Para cada motorista candidato      │
│                                          │
│  POST /v1/location/drivers/:id/lock      │
│  Body: {"ride_id": "...", "ttl": 15}     │
│                                          │
│  ┌─────────────────────────────────┐     │
│  │ Redis: SET driver:X:lock NX EX  │     │
│  └────────┬────────────────────────┘     │
│           │                              │
│     ┌─────┴──────┐                       │
│     │  Success?  │                       │
│     └─────┬──────┘                       │
│           │                              │
│     ┌─────┴───────────────────┐          │
│     │ YES          │ NO       │          │
│     ▼              ▼          │          │
│  BREAK        CONTINUE        │          │
│  (lock ok)    (próximo)       │          │
│     │                         │          │
└─────┼─────────────────────────┘          │
      │                                    │
      ▼                                    │
┌──────────────────────────────────┐       │
│ Publish: matching.driver_offered │       │
│ → Notification Service → Driver  │       │
└──────────────────────────────────┘       │
      │                                    │
      │ 3. Aguarda resposta (20s)          │
      ▼                                    │
┌──────────────────┐                       │
│ Driver responde? │                       │
└────┬─────────────┘                       │
     │                                     │
  ┌──┴────────┐                            │
  │ ACCEPT    │ REJECT/TIMEOUT             │
  ▼           ▼                            │
DELETE     DELETE                          │
 lock       lock                           │
  │          │                             │
  │          └─────────────────────────────┘
  │             (retry com próximo driver)
  ▼
RIDE MATCHED
```

**Redis é ideal porque:**
- Totalmente in-memory (latência sub-milissegundo)
- Operações atômicas nativas (`NX`, `EX`)
- Pode escalar horizontalmente (Redis Cluster)
- TTL nativo para expiração automática
- Separação de concerns (cache/locks vs dados transacionais)

---

## 2. Arquitetura RabbitMQ: Exchanges, Bindings e Queues

### 2.1. Conceitos Fundamentais

No RabbitMQ, a comunicação segue o padrão **Producer → Exchange → Binding → Queue → Consumer**:

```
┌──────────┐         ┌──────────┐         ┌─────────┐         ┌──────────┐
│ Producer │ ────→   │ Exchange │ ────→   │ Binding │ ────→   │  Queue   │ ────→ Consumer
└──────────┘  pub    └──────────┘  route  └─────────┘  bind   └──────────┘  sub
              (routing key)         (pattern match)            (buffer)
```

**Exchange:**
- Recebe mensagens dos producers
- Decide para qual(is) queue(s) rotear baseado em regras
- Tipos: direct, fanout, topic, headers

**Binding:**
- Conexão entre Exchange e Queue
- Define padrão de roteamento (routing key pattern)
- Uma exchange pode ter N bindings para diferentes queues

**Queue:**
- Buffer de mensagens
- Persiste mensagens até serem consumidas
- Um consumer (ou grupo) consome de uma queue

### 2.2. Estrutura do Sistema de Ride-Sharing

Utilizaremos **Topic Exchanges** para flexibilidade de roteamento com padrões hierárquicos.

#### Exchange 1: `rides.events`

**Tipo:** Topic Exchange
**Descrição:** Eventos relacionados ao ciclo de vida das corridas

**Routing Keys:**
- `ride.requested` - Passageiro solicitou corrida
- `ride.matched` - Corrida pareada com motorista
- `ride.accepted` - Motorista confirmou aceite
- `ride.started` - Corrida iniciada (pickup completo)
- `ride.completed` - Corrida finalizada
- `ride.cancelled` - Corrida cancelada (por passageiro ou motorista)

**Bindings e Queues:**

```
rides.events (Exchange)
    │
    ├─ Binding: "ride.requested" 
    │     └─→ Queue: matching_service_queue
    │           └─→ Consumer: Matching Service
    │
    ├─ Binding: "ride.matched"
    │     └─→ Queue: notification_rides_queue
    │           └─→ Consumer: Notification Service
    │
    ├─ Binding: "ride.started"
    │     └─→ Queue: notification_rides_queue
    │           └─→ consumer: Notification Service
    │
    ├─ Binding: "ride.completed"
    │     └─→ Queue: notification_rides_queue
    │           └─→ consumer: Notification Service
    └─ 
```

**Explicação:**
- **matching_service_queue**: Processa novas solicitações de corrida para iniciar matching
- **notification_rides_queue**: Notifica passageiros sobre mudanças de status
- **billing_events_queue**: Calcula cobrança (start/end da corrida)
- **analytics_events_queue**: Coleta métricas de negócio
- **audit_log_queue**: Registra todos os eventos de corrida para auditoria (usa wildcard `ride.*`)

#### Exchange 2: `matching.events`

**Tipo:** Topic Exchange
**Descrição:** Eventos do processo de matching entre passageiro e motorista

**Routing Keys:**
- `matching.candidate_found` - Candidato identificado (lock adquirido)
- `matching.offer_sent` - Oferta enviada ao motorista
- `matching.driver_accepted` - Motorista aceitou
- `matching.driver_rejected` - Motorista recusou
- `matching.driver_timeout` - Motorista não respondeu (20s)
- `matching.retry` - Tentando próximo candidato
- `matching.timeout` - Não foi possível encontrar nenhum motorista

**Bindings e Queues:**

```
matching.events (Exchange)
    │
    ├─ Binding: "matching.candidate_found"
    │     ├─→ Queue: notification_matching_queue
    │     │     └─→ Consumer: Notification Service
    │     │           (envia push/websocket para motorista)
    │     └─→ Queue: ride_updates_queue
    │           └─→ Consumer: Notification Service
    │                 (atualiza status no Postgres)
    │
    ├─ Binding: "matching.driver_accepted"
    │     └─→ Queue: ride_updates_queue
    │           └─→ Consumer: Ride Service
    │                 (atualiza status no Postgres)
    │                 (envia push/websocket para passageiro)
    │
    ├─ Binding: "matching.driver_rejected"
    │     └─→ Queue: matching_retry_queue
    │           └─→ Consumer: Matching Service
    │                 (tenta próximo motorista)
    │
    ├─ Binding: "matching.driver_timeout"
    │     └─→ Queue: matching_retry_queue
    │           └─→ Consumer: Matching Service
    │                 (tenta próximo motorista)
    │
    ├─ Binding: "matching.timeout"
    │     └─→ Queue: matching_timeout_queue
    │           └─→ Consumer: Matching Service
    │                 (atualiza status no Postgres)
    │                 (sinaliza que não foi possível encontrar nenhum motorista para o passageiro)
    └─ 
```

**Explicação:**
- **notification_matching_queue**: Dispara notificações para motoristas sobre novas ofertas
- **ride_updates_queue**: Atualiza estado da corrida no banco de dados
- **matching_retry_queue**: Reprocessa matching quando motorista recusa/timeout

#### Exchange 3: `location.events`

**Tipo:** Topic Exchange
**Descrição:** Status de motoristas

**Routing Keys:**
- `location.driver.updated` - Ping de GPS recebido
- `location.driver.online` - Motorista ficou disponível
- `location.driver.offline` - Motorista ficou indisponível
- `location.driver.idle` - Motorista disponível sem corrida (5+ min)

**Bindings e Queues:**

```
location.events (Exchange)
    │
    ├─ Binding: "location.driver.updated"
    │     ├─→ Queue: tracking_realtime_queue
    │     │     └─→ Consumer: Notification Service
    │     │           (envia posição para passageiro via WebSocket)
    │
    ├─ Binding: "location.driver.online"
    │     └─→ Queue: matching_availability_queue
    │           └─→ Consumer: Matching Service
    │                 (atualiza pool de motoristas disponíveis)
    │
    ├─ Binding: "location.driver.offline"
    │     └─→ Queue: matching_availability_queue
    │           └─→ Consumer: Matching Service
    │                 (atualiza pool de motoristas disponíveis)
    └─
```

**Explicação:**
- **tracking_realtime_queue**: Transmite posição do motorista para passageiro em corrida ativa
- **matching_availability_queue**: Mantém cache local de motoristas online no Matching Service

### 2.3. Configuração de Queues

Todas as queues devem ter as seguintes configurações:

**Durabilidade:**
```golang
{
  durable: true,              // Queue sobrevive a restart do RabbitMQ
  autoDelete: false,          // Não deleta quando último consumer desconecta
  messageTtl: 300000,         // Mensagens expiram após 5 minutos
  maxLength: 10000,           // Limite de mensagens na fila
  deadLetterExchange: "dlx",  // Mensagens expiradas vão para DLX
  deadLetterRoutingKey: "dlq.{service_name}"
}
```

**Dead Letter Exchange (DLX):**
Mensagens que falharam 3x ou expiraram vão para filas especiais:

```
dlx (Exchange - fanout)
    └─→ Queue: dead_letter_queue
          └─→ Consumer: Manual Review
```

### 2.4. Exemplos Completos de Fluxos

#### Cenário 1: Motorista Aceita a Corrida ✅

```
1. Ride Service publica mensagem:
   Exchange: rides.events
   Routing Key: ride.requested
   Body: {
     "ride_id": "ride-789",
     "rider_id": "rider-456",
     "pickup": {"lat": -23.55, "lng": -46.63}
   }

2. RabbitMQ roteia para:
   ✓ matching_service_queue (binding: "ride.requested")

3. Matching Service consome evento:
   - Chama GET /v1/location/drivers/nearby
   - Recebe lista: [driver-101, driver-102, driver-103]
   - Tenta adquirir lock do primeiro candidato

4. Matching Service adquire lock:
   POST /v1/location/drivers/driver-101/lock
   Body: {"ride_id": "ride-789", "ttl_seconds": 15}
   Response: {"locked": true, "expires_at": "2026-02-21T10:05:15Z"}

5. Matching Service publica:
   Exchange: matching.events
   Routing Key: matching.candidate_found
   Body: {
     "ride_id": "ride-789",
     "driver_id": "driver-101",
     "expires_at": "2026-02-21T10:05:35Z"
   }

6. RabbitMQ roteia para:
   ✓ notification_matching_queue (binding: "matching.candidate_found")

7. Notification Service consome → envia push/WebSocket para motorista:
   "Nova corrida disponível! R$ 25,50 - 1.2km de distância"

8. Motorista ACEITA → App chama:
   PATCH /rides/ride-789/accept
   Body: {"driver_id": "driver-101"}

9. Ride Service atualiza Postgres:
   UPDATE rides SET status = 'MATCHED', driver_id = 'driver-101'

10. Ride Service publica:
    Exchange: matching.events
    Routing Key: matching.driver_accepted
    Body: {
      "ride_id": "ride-789",
      "driver_id": "driver-101",
      "accepted_at": "2026-02-21T10:05:10Z"
    }

11. RabbitMQ roteia para:
    ✓ ride_updates_queue (binding: "matching.driver_accepted")

12. Ride Service consome e libera lock: // Acho que isso aqui só deve acontecer quando a corrida é finalizada ou cancelada
    DELETE /v1/location/drivers/driver-101/lock

13. Ride Service publica confirmação:
    Exchange: rides.events
    Routing Key: ride.matched
    Body: {
      "ride_id": "ride-789",
      "rider_id": "rider-456",
      "driver_id": "driver-101"
    }

14. Notification Service notifica passageiro:
    "Motorista encontrado! João está a caminho."

✅ SUCESSO: Ride matched em ~10 segundos
```

---

#### Cenário 2: Motorista Recusa a Corrida 🚫

```
1-7. [Mesmos passos do Cenário 1]
     Oferta enviada para driver-101

8. Motorista RECUSA → App chama:
   PATCH /rides/ride-789/reject
   Body: {
     "driver_id": "driver-101",
   }

9. Ride Service publica:
   Exchange: matching.events
   Routing Key: matching.driver_rejected
   Body: {
     "ride_id": "ride-789",
     "driver_id": "driver-101",
     "rejected_at": "2026-02-21T10:05:08Z",
   }

10. RabbitMQ roteia para:
    ✓ matching_retry_queue (binding: "matching.driver_rejected")

11. Ride Service libera lock:
    DELETE /v1/location/drivers/driver-101/lock
    → driver-101 volta ao pool de disponíveis

12. Matching Service consome evento da retry queue:
    - Pega próximo candidato da lista original: driver-102
    - Tenta adquirir lock

13. Matching Service adquire lock do driver-102:
    POST /v1/location/drivers/driver-102/lock
    Response: {"locked": true}

14. Matching Service publica:
    Exchange: matching.events
    Routing Key: matching.candidate_found
    Body: {
      "ride_id": "ride-789",
      "driver_id": "driver-102",
      "attempt": 2,
      "previous_rejections": ["driver-101"]
    }

15. Notification Service envia oferta para driver-102

16. Driver-102 ACEITA:
    → Segue fluxo do Cenário 1 (steps 8-14)

✅ SUCESSO: Ride matched após 1 retry (~25 segundos total)
```

---

#### Cenário 3: Motorista Não Responde (Timeout) ⏱️

```
1-7. [Mesmos passos do Cenário 1]
     Oferta enviada para driver-101 às 10:05:00

8. [20 segundos se passam sem resposta do motorista]
   Temporal Workflow detecta timeout às 10:05:20

9. Temporal publica:
   Exchange: matching.events
   Routing Key: matching.timeout
   Body: {
     "ride_id": "ride-789",
     "driver_id": "driver-101",
     "timeout_at": "2026-02-21T10:05:20Z",
     "ttl_expired": false
   }

10. RabbitMQ roteia para:
    ✓ matching_retry_queue (binding: "matching.timeout")

11. [Paralelamente] Redis expira lock automaticamente:
    Key: driver:driver-101:lock (TTL=15s expira às 10:05:15)
    → driver-101 automaticamente disponível

12. Matching Service consome timeout event:
    - Verifica que já passou 20s
    - Pega próximo candidato: driver-102

13. Matching Service tenta lock em driver-102:
    POST /v1/location/drivers/driver-102/lock
    Response: {"locked": true}

14. Matching Service publica:
    Exchange: matching.events
    Routing Key: matching.candidate_found
    Body: {
      "ride_id": "ride-789",
      "driver_id": "driver-102",
      "attempt": 2,
      "previous_timeout": ["driver-101"]
    }

15. Notification Service envia oferta para driver-102

16. Driver-102 ACEITA imediatamente:
    → Segue fluxo do Cenário 1 (steps 8-14)

✅ SUCESSO: Ride matched após timeout (~45 segundos total)

Observação: Se driver-102 TAMBÉM der timeout:
→ Repete processo com driver-103
→ Continua até matching.max_attempts (padrão: 10 tentativas)
→ Se esgotar tentativas → vai para Cenário 4
```

---

#### Cenário 4: Nenhum Motorista Encontrado (5 minutos) ❌

```
1-3. [Mesmos passos do Cenário 1]
     Matching Service busca candidatos

4. Loop de tentativas começa às 10:00:00:

   10:00:00 - Tenta driver-101 → TIMEOUT (20s)
   10:00:20 - Tenta driver-102 → REJECTED (5s)
   10:00:25 - Tenta driver-103 → TIMEOUT (20s)
   10:00:45 - Tenta driver-104 → REJECTED (3s)
   10:00:48 - Tenta driver-105 → TIMEOUT (20s)
   10:01:08 - Tenta driver-106 → REJECTED (7s)
   10:01:15 - Tenta driver-107 → TIMEOUT (20s)
   10:01:35 - Tenta driver-108 → REJECTED (4s)
   10:01:39 - Tenta driver-109 → TIMEOUT (20s)
   10:01:59 - Tenta driver-110 → REJECTED (2s)

   10:02:01 - Lista de candidatos esgotada (10 tentativas)

5. Matching Service busca novos candidatos:
   GET /v1/location/drivers/nearby (2ª busca)
   → Retorna lista vazia (nenhum motorista disponível no raio)

6. Matching Service aguarda 30 segundos e tenta novamente:
   10:02:31 - 3ª busca → lista vazia
   10:03:01 - 4ª busca → lista vazia
   10:03:31 - 5ª busca → lista vazia
   10:04:01 - 6ª busca → lista vazia
   10:04:31 - 7ª busca → lista vazia

7. [5 minutos totais se passaram desde 10:00:00]
   Temporal Workflow atinge timeout global (5 minutos)

8. Matching Service publica:
   Exchange: matching.events
   Routing Key: matching.no_drivers_available
   Body: {
     "ride_id": "ride-789",
     "rider_id": "rider-456",
     "started_at": "2026-02-21T10:00:00Z",
     "timeout_at": "2026-02-21T10:05:00Z",
     "total_attempts": 10,
     "rejected_drivers": ["driver-102", "driver-104", "driver-106", "driver-108", "driver-110"],
     "timeout_drivers": ["driver-101", "driver-103", "driver-105", "driver-107", "driver-109"],
     "search_radius_km": 5,
     "reason": "no_drivers_in_area"
   }

9. RabbitMQ roteia para:
   ✓ notification_matching_queue (binding: "matching.*")
   ✓ analytics_matching_queue (binding: "matching.*")

10. Ride Service atualiza status:
    UPDATE rides SET status = 'CANCELLED',
    cancelled_reason = 'no_drivers_available',
    cancelled_at = NOW()

11. Ride Service publica:
    Exchange: rides.events
    Routing Key: ride.cancelled
    Body: {
      "ride_id": "ride-789",
      "rider_id": "rider-456",
      "cancelled_by": "system",
      "reason": "no_drivers_available",
      "cancelled_at": "2026-02-21T10:05:00Z"
    }

12. Notification Service notifica passageiro:
    Push Notification:
    {
      "title": "Nenhum motorista disponível",
      "message": "Não encontramos motoristas na sua região no momento. Tente novamente em alguns minutos ou aumente o raio de busca.",
      "actions": [
        "retry_same_location",
        "increase_radius",
        "schedule_later"
      ]
    }

13. Analytics Service registra métrica:
    - Incrementa contador: failed_matches_no_drivers
    - Registra horário/localização para análise de demanda
    - Dispara alerta se taxa de falha > 10% na região

❌ FALHA: Ride cancelled - nenhum motorista encontrado em 5 minutos
```

---

#### Resumo Visual: Árvore de Decisão

```
                    [Ride Requested]
                          │
                          ▼
              ┌───────────────────────┐
              │  Busca Candidatos     │
              └───────────┬───────────┘
                          │
                ┌─────────┴─────────┐
                │ Encontrou drivers? │
                └─────────┬──────────┘
                          │
            ┌─────────────┼─────────────┐
            │ SIM                       │ NÃO
            ▼                           ▼
    ┌──────────────┐            ┌─────────────┐
    │ Tenta Lock   │            │ Aguarda 30s │
    └──────┬───────┘            └──────┬──────┘
           │                            │
    ┌──────┴───────┐                   │
    │ Lock obtido? │                   │
    └──────┬───────┘                   │
           │                            │
     ┌─────┴─────┐                     │
     │ SIM       │ NÃO → Próximo       │
     ▼           ▼       candidato     │
  Envia      Próximo                   │
  Oferta     Driver                    │
     │                                 │
     ▼                                 │
┌─────────────┐                        │
│ Resposta?   │                        │
└─────┬───────┘                        │
      │                                │
  ┌───┴───┬────────┬──────┐           │
  │       │        │      │           │
ACEITA  RECUSA  TIMEOUT  [5 min]      │
  │       │        │      │           │
  ▼       │        │      ▼           │
✅       │        │    Cenário 4 ◄────┘
MATCHED   │        │      │
          │        │      ▼
          │        │    Notifica
          │        │    Passageiro
          │        │      ❌
          │        │
          └────────┴─→ Próximo
                       Candidato
                       (retry)
```

---

#### Configuração de Timeouts e Limites

```javascript
// Configurações do Matching Service
{
  "driver_response_timeout": 20,        // segundos
  "lock_ttl": 15,                       // segundos (expira antes do timeout)
  "max_attempts_per_search": 10,        // tentativas antes de buscar novamente
  "search_retry_interval": 30,          // segundos entre buscas
  "global_matching_timeout": 300,       // 5 minutos - timeout total
  "initial_search_radius": 5000,        // 5km em metros
  "max_search_radius": 15000,           // 15km em metros
  "radius_increment": 2500              // aumenta 2.5km por busca
}
```

### 2.5. Vantagens da Arquitetura Topic Exchange

**1. Desacoplamento:**
- Producers não conhecem consumers
- Novos serviços podem "escutar" eventos existentes sem mudanças

**2. Flexibilidade de Roteamento:**
```javascript
// Um consumer pode fazer bind com padrões complexos:
"ride.*"                    // Todos eventos de ride
"matching.driver.*"         // Apenas respostas de motoristas
"*.completed"               // Eventos de conclusão de qualquer tipo
"#"                         // Tudo (útil para logging)
```

**3. Múltiplos Consumers:**
- Mesmo evento pode ser processado por N serviços
- Cada um com seu propósito (notificação, billing, analytics)

**4. Evolução Sem Quebra:**
- Adicionar novo routing key não afeta consumers existentes
- Novos consumers podem consumir eventos históricos (se persistidos)

**5. Resiliência:**
- Dead Letter Queues capturam falhas
- Retry automático com exponential backoff
- Mensagens não são perdidas mesmo se consumer estiver offline

---

## 3. Resumo Visual

### Visão Geral da Arquitetura de Mensageria

```
┌──────────────────────────────────────────────────────────────┐
│                         PRODUCERS                             │
│  Ride Service  │  Matching Service  │  Location Service      │
└────────┬───────────────┬──────────────────────┬──────────────┘
         │               │                      │
         │ publish       │ publish              │ publish
         │               │                      │
         ▼               ▼                      ▼
┌─────────────────────────────────────────────────────────────┐
│                      RABBITMQ BROKER                         │
│                                                              │
│  ┌───────────────┐  ┌─────────────────┐  ┌───────────────┐ │
│  │ rides.events  │  │ matching.events │  │location.events│ │
│  │   (topic)     │  │     (topic)     │  │    (topic)    │ │
│  └───────┬───────┘  └────────┬────────┘  └───────┬───────┘ │
│          │                   │                   │          │
│          │ bindings          │ bindings          │ bindings │
│          ▼                   ▼                   ▼          │
│  ┌────────────────────────────────────────────────────────┐ │
│  │                      QUEUES                            │ │
│  │  matching_service_queue  │  notification_*_queue       │ │
│  │  ride_updates_queue      │                             │ │
│  │  matching_retry_queue    │                             │ │
│  └────────────────────────────────────────────────────────┘ │
└─────────────────────────────────────────────────────────────┘
         │               │                      │
         │ consume       │ consume              │ consume
         ▼               ▼                      ▼
┌──────────────────────────────────────────────────────────────┐
│                        CONSUMERS                              │
│  Matching Service  │  Notification Service  │  Ride Service  │
│  Analytics Service │  Audit Service         │  Billing Svc   │
└──────────────────────────────────────────────────────────────┘
```

---

## 4. Referências

- [Redis Distributed Locks](https://redis.io/docs/manual/patterns/distributed-locks/)
- [RabbitMQ Topic Exchange](https://www.rabbitmq.com/tutorials/tutorial-five-python.html)
- [Event-Driven Architecture Patterns](https://martinfowler.com/articles/201701-event-driven.html)
