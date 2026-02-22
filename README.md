# Ride Sharing with Golang and Event-Driven Microservices

Plataforma de mobilidade urbana focada na conexão em tempo real entre passageiros e motoristas. O projeto implementa um motor de pareamento (matching) de ultra-baixa latência e alta consistência, utilizando uma arquitetura orientada a eventos (Event-Driven) sobre microsserviços.

## 🚀 Stack Tecnológica

* **Linguagem:** Golang 1.26
* **Banco de Dados:** PostgreSQL (com PostGIS) e Redis
* **Mensageria:** RabbitMQ
* **Infraestrutura/Orquestração:** Kubernetes (Minikube), Docker e Tilt

## 🏗️ Arquitetura (Microsserviços)

O ecossistema é dividido nos seguintes domínios:
* **API Gateway:** Ponto único de entrada responsável por roteamento e segurança.
* **User Service:** Gestão de passageiros e motoristas.
* **Location Service:** Ingestão assíncrona de telemetria de GPS e indexação geoespacial no Redis.
* **Ride Service:** Orquestrador do ciclo de vida das corridas e cálculo de tarifas.
* **Matching Service:** Cérebro do pareamento utilizando *distributed locking* para evitar conflitos.
* **Notification Service:** Comunicação em tempo real com os aplicativos clientes via WebSockets.

## 📋 Pré-requisitos

Para executar o projeto localmente, você precisará das seguintes ferramentas instaladas:
* Docker
* Kubectl
* Minikube
* Tilt
* Golang 1.26

## 🛠️ Guia de Instalação de Ferramentas (WSL / Linux)

Se você está configurando o ambiente do zero no WSL, siga os passos abaixo:

1. Verificar Docker
```bash
docker version
```

2. Instalar Kubectl
```bash
sudo apt update
sudo apt install -y curl
curl -LO "[https://dl.k8s.io/release/$(curl](https://dl.k8s.io/release/$(curl) -Ls [https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl](https://dl.k8s.io/release/stable.txt)/bin/linux/amd64/kubectl)"
chmod +x kubectl
sudo mv kubectl /usr/local/bin/
kubectl version --client
```

3. Instalar Minikube
```bash
curl -LO [https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64](https://storage.googleapis.com/minikube/releases/latest/minikube-linux-amd64)
sudo install minikube-linux-amd64 /usr/local/bin/minikube
minikube version
```

4. Instalar Tilt
```bash
curl -fsSL [https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh](https://raw.githubusercontent.com/tilt-dev/tilt/master/scripts/install.sh) | bash
tilt version
```

## 💻 Como Executar o Projeto

O ambiente de desenvolvimento é automatizado utilizando `Make` e `Tilt`.

1. **Inicie o cluster Minikube:**
```bash
make minikube-start
```

2. **Suba a infraestrutura e os microsserviços:*
```bash
make dev-up
```

O Tilt iniciará todos os recursos (Postgres, Redis, RabbitMQ) e fará o live-reload dos serviços em Go.

3. Para encerrar e limpar o ambiente:
```bash
make dev-down
```
