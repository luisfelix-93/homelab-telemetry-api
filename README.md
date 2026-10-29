# Homelab Telemetry API

API de observabilidade e telemetria de infraestrutura em tempo real desenvolvida em **Go**, projetada para monitoramento de hosts bare-metal (nós de computação) e microserviços em ambientes homelab. 

Projetada com foco em performance, baixo consumo de recursos e desacoplamento (Clean Architecture), provendo endpoints REST padronizados e interface para emulação de terminal interativo integrado a portfólios e dashboards web.

---

## 🚀 Funcionalidades

- **Coleta de Métricas do Host (Node Telemetry)**: Monitoramento em tempo real de hardware e sistema operacional (cores de CPU, porcentagem de uso de processamento, memória RAM total/usada e tempo de atividade/uptime) utilizando `gopsutil`.
- **Probes de Saúde HTTP (Service Telemetry)**: Verificação contínua e assíncrona de status e latência de microserviços externos e internos com timeout configurável.
- **Worker em Background**: Rotina de telemetria assíncrona em goroutine dedicada, compilando snapshots consolidados da infraestrutura periodicamente.
- **Cache In-Memory Thread-Safe**: Repositório de snapshots em memória utilizando leituras e escritas concorrentes seguras (`sync.RWMutex`), garantindo latência de resposta na faixa de microssegundos sem gargalos de banco de dados.
- **Parser de Comandos de Terminal Web**: Processador de comandos via endpoint POST com saídas duplas (texto pré-formatado estilo shell e payload JSON estruturado).
- **Graceful Shutdown**: Encerramento seguro de goroutines e conexões HTTP ativas mediante sinais de terminação do sistema operacional (`SIGINT`, `SIGTERM`).
- **Suporte Nativo a CORS & Middlewares**: Pronto para consumo direto a partir de SPAs (React, Vue, Vite, etc.) com middlewares de Request ID, Real IP, Logger e Recoverer (`chi/v5`).

---

## 🏗️ Arquitetura

O projeto adota padrões inspirados em **Clean Architecture / Arquitetura Hexagonal**:

```
homelab-telemetry-api/
├── cmd/
│   └── api/
│       └── main.go                  # Ponto de entrada, injeção de dependências e graceful shutdown
├── internal/
│   ├── config/                     # Utilitários de carregamento de ambiente
│   ├── domain/                     # Entidades de negócio, contratos de repositório e interfaces
│   ├── infra/
│   │   ├── cache/                  # Implementação de repositório in-memory thread-safe
│   │   └── collector/              # Coletores de sistema (gopsutil) e sondas HTTP
│   ├── service/                    # Worker periódico e analisador de comandos (Command Parser)
│   └── transport/
│       └── http/                   # Roteador Chi, middlewares e controladores HTTP
├── .env.example                    # Exemplo de variáveis de ambiente
├── go.mod
└── go.sum
```

---

## ⚙️ Variáveis de Ambiente

Crie um arquivo `.env` na raiz da aplicação com base no `.env.example`:

```bash
cp .env.example .env
```

| Variável | Descrição | Exemplo / Padrão |
| :--- | :--- | :--- |
| `PORT` | Porta onde o servidor HTTP escuta requisições | `8080` |
| `SERVICE_TARGETS` | Lista de serviços monitorados no formato `nome=url` separados por vírgula | `mcp-server=https://example.com/health,agent=https://agent.example.com/api/health` |

---

## 🛠️ Começando (Quick Start)

### Pré-requisitos

- [Go](https://golang.org/dl/) 1.22 ou superior instalado.

### 1. Clonar e Instalar Dependências

```bash
git clone <url-do-repositorio>
cd homelab-telemetry-api
go mod download
```

### 2. Configurar o Ambiente

Configure o arquivo `.env` com a porta e os alvos de sondagem desejados.

### 3. Executar a Aplicação

```bash
go run cmd/api/main.go
```

Saída esperada no terminal:
```log
2026/10/27 21:00:00 [Main] Starting Homelab Telemetry API...
2026/10/27 21:00:00 [Worker] background collector started (interval: 20s)
2026/10/27 21:00:00 [HTTP Server] listening on port 8080
```

---

## 📡 Endpoints da API

A API expõe suas rotas sob o prefixo `/api/v1`:

### 1. Healthcheck

Verifica se a API está online e respondendo.

- **Método**: `GET`
- **Rota**: `/api/v1/health`
- **Resposta**:
  ```json
  {
    "status": "ok"
  }
  ```

---

### 2. Visão Geral do Cluster (Overview)

Retorna o snapshot agregado do cluster, consolidando consumo total de CPU/RAM, nós ativos e serviços saudáveis.

- **Método**: `GET`
- **Rota**: `/api/v1/telemetry/overview`
- **Resposta**:
  ```json
  {
    "cluster_name": "homelab-alpha",
    "environment": "bare-metal-edge",
    "uptime_seconds": 124580,
    "last_updated": "2026-10-27T21:40:00Z",
    "resources": {
      "cpu_usage_percent": 14.8,
      "memory_used_bytes": 8589934592,
      "memory_total_bytes": 34359738368,
      "memory_usage_percent": 25.0
    },
    "total_nodes": 1,
    "healthy_nodes": 1,
    "total_services": 2,
    "healthy_services": 2
  }
  ```

---

### 3. Métricas de Serviços (HTTP Probes)

Retorna o status de saúde e a latência de cada microserviço cadastrado via `SERVICE_TARGETS`.

- **Método**: `GET`
- **Rota**: `/api/v1/telemetry/services`
- **Resposta**:
  ```json
  [
    {
      "name": "mcp-server",
      "status": "UP",
      "latency_ms": 32,
      "last_check": "2026-10-27T21:40:00Z",
      "message": "HTTP 200 OK"
    },
    {
      "name": "support-agent",
      "status": "UP",
      "latency_ms": 58,
      "last_check": "2026-10-27T21:40:00Z",
      "message": "HTTP 200 OK"
    }
  ]
  ```

---

### 4. Métricas de Nós (Hardware Nodes)

Retorna métricas de hardware de cada máquina/nó que compõe o homelab.

- **Método**: `GET`
- **Rota**: `/api/v1/telemetry/nodes`
- **Resposta**:
  ```json
  [
    {
      "id": "lffilho-server",
      "role": "control-plane",
      "status": "Ready",
      "cpu_cores": 16,
      "cpu_percent": 12.5,
      "memory_total_mb": 32768,
      "memory_used_mb": 8192,
      "uptime_seconds": 124580
    }
  ]
  ```

---

### 5. Terminal Interativo

Permite enviar comandos de emulação de terminal web e receber respostas formatadas em modo texto monoespaçado e em dados estruturados.

- **Método**: `POST`
- **Rota**: `/api/v1/cmd`
- **Payload de Requisição**:
  ```json
  {
    "command": "status"
  }
  ```
- **Resposta**:
  ```json
  {
    "command": "status",
    "timestamp": "2026-10-27T21:40:00Z",
    "output_text": "Cluster: homelab-alpha (bare-metal-edge)\nUptime: 124580s | Nodes: 1/1 Healthy | Services: 2/2 Healthy\nCPU Usage: 14.8% | Memory Usage: 25.0%",
    "data": {
      "cluster_name": "homelab-alpha",
      "environment": "bare-metal-edge"
    }
  }
  ```

#### Comandos Disponíveis no Terminal

| Comando | Descrição |
| :--- | :--- |
| `status` | Exibe o resumo do cluster, nós ativos, serviços saudáveis e consumo geral |
| `services` | Tabela formatada contendo `NAME`, `STATUS`, `LATENCY` e `LAST CHECK` dos serviços |
| `nodes` | Tabela de hardware dos nós com `ID`, `ROLE`, `STATUS`, `CPU%` e `RAM (USED/TOT)` |
| `help` | Exibe a lista de comandos suportados |
| `clear` | Retorna buffer vazio para limpeza da tela do terminal |

---

## 🧪 Testes

A suíte de testes unitários e de integração cobre coletores, analisador de comandos, parsers de alvos, cache in-memory e handlers HTTP:

```bash
# Executar todos os testes
go test -v ./...

# Executar testes com detecção de data race
go test -race ./...
```

---

## 🚀 Integração e Entrega Contínua (CI/CD)

O repositório possui pipelines automatizados configurados via **GitHub Actions**:

1. **Validação Contínua (`ci.yml`)**:
   - Disparado em qualquer Pull Request ou push em branches de trabalho.
   - Executa a suíte completa de testes com detecção de concorrência (`go test -v -race ./...`).

2. **Release e Publicação Docker (`release.yml`)**:
   - Disparado automaticamente após o merge na branch `main`.
   - Executa os testes automatizados como portão de qualidade.
   - Analisa os commits, calcula o versionamento semântico (SemVer) e gera a Git Tag + GitHub Release.
   - Constrói o contêiner estático da aplicação e publica no **Docker Hub** com múltiplas tags:
     - `:latest` - versão estável mais recente da `main`
     - `:vX.Y.Z` - versão semântica oficial da release (ex: `v0.1.0`)
     - `:YYYYmmDD` - tag diária no formato CalVer (ex: `20261029`)
     - `:YYYYmmDD-<sha>` - tag imutável com data e hash curto do commit Git (ex: `20261029-ccdf426`)

### Configuração de Secrets no GitHub

Para que o deploy no Docker Hub funcione, configure os seguintes segredos em **Settings > Secrets and variables > Actions**:

| Secret | Descrição |
| :--- | :--- |
| `DOCKERHUB_USERNAME` | Seu nome de usuário no Docker Hub |
| `DOCKERHUB_PASSWORD` | Sua senha de acesso ao Docker Hub |

---

## 📄 Licença

Este projeto é desenvolvido para fins de observabilidade de homelab e integração com portfólio pessoal. Distribuído sob a licença [MIT](LICENSE).
