package main

import (
	"context"
	"encoding/json"
	"flag"
	"fmt"
	"log"
	"net/http"
	"os"
	"os/signal"
	"syscall"
	"time"

	"github.com/redis/go-redis/v9"
	"go.uber.org/zap"

	"github.com/Aixtrade/TaskFlow/internal/config"
	grpcclient "github.com/Aixtrade/TaskFlow/internal/infrastructure/grpc"
	"github.com/Aixtrade/TaskFlow/internal/infrastructure/observability/logging"
	asynqqueue "github.com/Aixtrade/TaskFlow/internal/infrastructure/queue/asynq"
	"github.com/Aixtrade/TaskFlow/internal/worker"
	"github.com/Aixtrade/TaskFlow/internal/worker/handlers/demo"
	grpctask "github.com/Aixtrade/TaskFlow/internal/worker/handlers/grpc_task"
	"github.com/Aixtrade/TaskFlow/pkg/progress"
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

	// 初始化 Redis 客户端（用于进度发布）
	redisClient := redis.NewClient(&redis.Options{
		Addr:     cfg.Redis.Addr,
		Password: cfg.Redis.Password,
		DB:       cfg.Redis.DB,
	})
	defer redisClient.Close()

	// 创建进度发布器
	progressPublisher := progress.NewPublisher(redisClient, logger, progress.StreamOptions{
		MaxLen:      cfg.Progress.MaxLen,
		TTL:         cfg.Progress.TTL,
		ReadTimeout: cfg.Progress.ReadTimeout,
	})

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
				HealthCheckInterval: cfg.GRPCServices.Defaults.HealthCheckInterval,
				MaxRetries:          cfg.GRPCServices.Defaults.MaxRetries,
				RetryDelay:          cfg.GRPCServices.Defaults.RetryDelay,
			},
		}
		registry.Register(grpctask.NewHandler(logger, clientManager, grpcTaskConfig, progressPublisher))

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
	)

	registry.SetupServer(server)

	go func() {
		if err := server.Start(); err != nil {
			logger.Fatal("failed to start server", zap.Error(err))
		}
	}()

	var healthServer *http.Server
	if cfg.Server.Worker.Health.Enabled {
		healthMux := http.NewServeMux()
		healthMux.HandleFunc("/health", func(w http.ResponseWriter, r *http.Request) {
			status := "healthy"
			services := map[string]string{}

			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				services["redis"] = "unhealthy"
				status = "unhealthy"
			} else {
				services["redis"] = "healthy"
			}

			if clientManager != nil {
				for _, svc := range clientManager.GetHealthStatus() {
					name := fmt.Sprintf("grpc:%s", svc.Name)
					if svc.Healthy {
						services[name] = "healthy"
					} else {
						services[name] = "unhealthy"
						status = "unhealthy"
					}
				}
			}

			payload := map[string]interface{}{
				"status":    status,
				"timestamp": time.Now().UTC().Format(time.RFC3339),
				"services":  services,
			}
			if status != "healthy" {
				w.WriteHeader(http.StatusServiceUnavailable)
			}
			_ = json.NewEncoder(w).Encode(payload)
		})

		healthMux.HandleFunc("/ready", func(w http.ResponseWriter, r *http.Request) {
			ctx, cancel := context.WithTimeout(r.Context(), 3*time.Second)
			defer cancel()

			if err := redisClient.Ping(ctx).Err(); err != nil {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"status": "not ready",
					"reason": "redis unavailable",
				})
				return
			}

			if clientManager != nil && len(clientManager.UnhealthyServices()) > 0 {
				w.WriteHeader(http.StatusServiceUnavailable)
				_ = json.NewEncoder(w).Encode(map[string]string{
					"status": "not ready",
					"reason": "grpc services unavailable",
				})
				return
			}

			_ = json.NewEncoder(w).Encode(map[string]string{"status": "ready"})
		})

		healthMux.HandleFunc("/live", func(w http.ResponseWriter, r *http.Request) {
			_ = json.NewEncoder(w).Encode(map[string]string{"status": "alive"})
		})

		addr := fmt.Sprintf("%s:%d", cfg.Server.Worker.Health.Host, cfg.Server.Worker.Health.Port)
		healthServer = &http.Server{
			Addr:              addr,
			Handler:           healthMux,
			ReadHeaderTimeout: 5 * time.Second,
		}

		go func() {
			logger.Info("starting worker health server", zap.String("addr", addr))
			if err := healthServer.ListenAndServe(); err != nil && err != http.ErrServerClosed {
				logger.Fatal("failed to start worker health server", zap.Error(err))
			}
		}()
	}

	quit := make(chan os.Signal, 1)
	signal.Notify(quit, syscall.SIGINT, syscall.SIGTERM)
	<-quit

	logger.Info("shutting down server...")
	if healthServer != nil {
		ctx, cancel := context.WithTimeout(context.Background(), 5*time.Second)
		if err := healthServer.Shutdown(ctx); err != nil {
			logger.Error("failed to shutdown health server", zap.Error(err))
		}
		cancel()
	}
	server.Shutdown()
	logger.Info("server stopped")
}
