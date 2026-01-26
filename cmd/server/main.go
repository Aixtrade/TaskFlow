package main

import (
	"flag"
	"log"
	"os"
	"os/signal"
	"syscall"

	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/config"
	grpcclient "github.com/Aixtrade/TaskFlow/internal/infrastructure/grpc"
	"github.com/Aixtrade/TaskFlow/internal/infrastructure/observability/logging"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	"github.com/Aixtrade/TaskFlow/internal/worker"
	"github.com/Aixtrade/TaskFlow/internal/worker/handlers/demo"
	grpctask "github.com/Aixtrade/TaskFlow/internal/worker/handlers/grpc_task"
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

	// 初始化 gRPC 客户端管理器（如果启用）
	var clientManager *grpcclient.ClientManager
	if cfg.GRPCServices.Enabled && len(cfg.GRPCServices.Services) > 0 {
		clientConfigs := make(map[string]grpcclient.ClientConfig)
		for name, svcCfg := range cfg.GRPCServices.Services {
			clientConfigs[name] = grpcclient.ClientConfig{
				Address:             svcCfg.Address,
				Timeout:             svcCfg.Timeout,
				PoolSize:            svcCfg.PoolSize,
				HealthCheckInterval: svcCfg.HealthCheckInterval,
				MaxRetries:          svcCfg.MaxRetries,
				RetryDelay:          svcCfg.RetryDelay,
			}
		}

		var err error
		clientManager, err = grpcclient.NewClientManager(clientConfigs, logger)
		if err != nil {
			logger.Fatal("failed to create grpc client manager", zap.Error(err))
		}
		defer clientManager.Close()

		// 注册 gRPC handler
		grpcTaskConfig := grpctask.Config{
			Services: clientConfigs,
			Defaults: grpcclient.ClientConfig{
				Timeout:             cfg.GRPCServices.Defaults.Timeout,
				PoolSize:            cfg.GRPCServices.Defaults.PoolSize,
				HealthCheckInterval: cfg.GRPCServices.Defaults.HealthCheckInterval,
			},
		}
		registry.Register(grpctask.NewHandler(logger, clientManager, grpcTaskConfig))

		logger.Info("grpc services initialized",
			zap.Strings("services", clientManager.Services()),
		)
	}

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
