package main

import (
	"context"
	"errors"
	"io/fs"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"homelab-telemetry-api/internal/domain"
	"homelab-telemetry-api/internal/infra/cache"
	"homelab-telemetry-api/internal/infra/collector"
	"homelab-telemetry-api/internal/service"
	transporthttp "homelab-telemetry-api/internal/transport/http"

	"github.com/joho/godotenv"
)

func main() {
	log.Println("[Main] Starting Homelab Telemetry API...")

	if err := godotenv.Load(); err != nil && !errors.Is(err, fs.ErrNotExist) {
		log.Fatalf("[Main] failed to load .env: %v", err)
	}

	// 1. Contexto raiz da aplicação para controle de ciclo de vida
	ctx, cancel := context.WithCancel(context.Background())
	defer cancel()

	// 2. Inicialização do Repositório de Cache (In-Memory)
	repo := cache.NewMemorySnapshotRepository()

	// 3. Configuração dos Coletores (Targets do seu Homelab)
	// URLs definidas via SERVICE_TARGETS no formato "nome=url,nome=url" (veja .env.example)
	httpTargets, err := collector.ParseHTTPTargets(os.Getenv("SERVICE_TARGETS"))
	if err != nil {
		log.Fatalf("[Main] invalid SERVICE_TARGETS: %v", err)
	}
	if len(httpTargets) == 0 {
		log.Println("[Main] warning: SERVICE_TARGETS is empty, no HTTP services will be probed")
	}

	serviceCollector := collector.NewHTTPServiceCollector("homelab-http-probes", httpTargets, 3*time.Second)
	nodeCollector := collector.NewSystemNodeCollector("homelab-system-probe", "lffilho-server", "control-plane")

	// 4. Inicialização do Worker de Background
	// Roda a cada 20 segundos coletando métricas e atualizando o snapshot
	telemetryWorker := service.NewTelemetryWorker(
		repo,
		[]domain.ServiceCollector{serviceCollector},
		[]domain.NodeCollector{nodeCollector},
		"homelab-alpha",
		"bare-metal-edge",
	)

	// Dispara o worker em uma goroutine separada
	go telemetryWorker.Start(ctx, 20*time.Second)

	// 5. Inicialização do Command Parser e Handlers HTTP
	commandParser := service.NewCommandParser(repo)
	apiHandler := transporthttp.NewHandler(repo, commandParser)
	router := transporthttp.NewRouter(apiHandler)

	// 6. Configuração do Servidor HTTP
	port := getEnv("PORT", "8080")
	server := &http.Server{
		Addr:         ":" + port,
		Handler:      router,
		ReadTimeout:  15 * time.Second,
		WriteTimeout: 15 * time.Second,
		IdleTimeout:  60 * time.Second,
	}

	// Inicia o servidor HTTP em background
	go func() {
		log.Printf("[HTTP Server] listening on port %s", port)
		if err := server.ListenAndServe(); err != nil && !errors.Is(err, http.ErrServerClosed) {
			log.Fatalf("[HTTP Server] fatal error: %v", err)
		}
	}()

	// 7. Configuração do Graceful Shutdown (Escutando SIGINT e SIGTERM)
	quit := make(chan os.Signal, 1)
	signal.Notify(quit, os.Interrupt, syscall.SIGTERM)
	<-quit

	log.Println("[Main] Shutdown signal received, gracefully stopping services...")

	// Cancela o contexto raiz para encerrar o worker em background
	cancel()

	// Timeout de 10 segundos para fechar conexões HTTP pendentes
	shutdownCtx, shutdownCancel := context.WithTimeout(context.Background(), 10*time.Second)
	defer shutdownCancel()

	if err := server.Shutdown(shutdownCtx); err != nil {
		log.Printf("[HTTP Server] forced shutdown: %v", err)
	}

	log.Println("[Main] Application successfully stopped. Goodbye!")
}

func getEnv(key, fallback string) string {
	if value, ok := os.LookupEnv(key); ok && value != "" {
		return value
	}
	return fallback
}