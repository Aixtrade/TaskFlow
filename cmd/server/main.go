package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/config"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	"github.com/Aixtrade/TaskFlow/internal/infrastructure/observability/logging"
	"github.com/Aixtrade/TaskFlow/internal/worker"
	"github.com/Aixtrade/TaskFlow/internal/worker/handlers/demo"
)

func main() {
	configPath := flag.String("config", "", "path to config file")
	flag.Parse()

	cfg, err := config.Load(*configPath)
	if err != nil {
		log.Fatalf("failed to load config: %v", err)
	}

	logger, err := logging.NewLogger(&cfg.Logging)
	if err != nil {
		log.Fatalf("failed to create logger: %v", err)
	}
	defer logger.Sync()

	logger.Info("starting taskflow worker",
		zap.String("env", cfg.App.Env),
		zap.Int("concurrency", cfg.Server.Worker.Concurrency),
	)

	registry := worker.NewRegistry(logger)
	registry.Register(demo.NewHandler(logger))

	logger.Info("registered handlers", zap.Strings("types", registry.Types()))

	server, err := asynqqueue.NewServer(asynqqueue.ServerConfig{
		Redis:       &cfg.Redis,
		Queues:      cfg.Queues.ToMap(),
		Concurrency: cfg.Server.Worker.Concurrency,
		Logger:      logger,
	})
	if err != nil {
		logger.Fatal("failed to create server", zap.Error(err))
	}

	server.Use(
		worker.RecoveryMiddleware(logger),
		worker.LoggingMiddleware(logger),
		worker.MetricsMiddleware(),
	)

	registry.SetupServer(server)

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	server.Shutdown()
	logger.Info("server stopped")
}
