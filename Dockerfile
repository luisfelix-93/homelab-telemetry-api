# ------------------------------------------------------------------------------
# Build Stage
# ------------------------------------------------------------------------------
FROM golang:alpine AS builder

WORKDIR /app

# Instala certificados SSL e ferramentas necessárias para download de módulos
RUN apk add --no-cache git ca-certificates tzdata

# Cache de dependências do Go
COPY go.mod go.sum* ./
RUN go mod download

# Copia o código-fonte
COPY . .

# Compilação estática do binário Go
# -trimpath remove caminhos absolutos locais do executável
# -ldflags="-w -s" remove símbolos de depuração DWARF para reduzir o tamanho final
RUN CGO_ENABLED=0 GOOS=linux go build \
    -trimpath \
    -ldflags="-w -s" \
    -o /bin/homelab-telemetry-api \
    ./cmd/api

# ------------------------------------------------------------------------------
# Runtime Stage
# ------------------------------------------------------------------------------
FROM alpine:latest

WORKDIR /app

# Instala certificados CA para chamadas HTTPS dos coletores e dados de timezone
RUN apk --no-cache add ca-certificates tzdata \
    && addgroup -S appgroup \
    && adduser -S appuser -G appgroup

# Copia o binário estático da etapa de build
COPY --from=builder /bin/homelab-telemetry-api /app/homelab-telemetry-api

# Executa com usuário não-privilegiado por segurança
USER appuser:appgroup

# Configuração de porta padrão
ENV PORT=8080
EXPOSE 8080

# Checagem de saúde da aplicação usando o endpoint existente
HEALTHCHECK --interval=30s --timeout=5s --start-period=5s --retries=3 \
    CMD wget -qO- http://localhost:${PORT}/api/v1/health || exit 1

ENTRYPOINT ["/app/homelab-telemetry-api"]
