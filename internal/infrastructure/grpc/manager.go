package grpc

import (
	"fmt"
	"sync"

	"go.uber.org/zap"
)

// ClientManager 管理多个 gRPC 服务客户端
type ClientManager struct {
	clients map[string]*StreamingGRPCClient
	configs map[string]ClientConfig
	mu      sync.RWMutex
	logger  *zap.Logger
}

// NewClientManager 创建客户端管理器
func NewClientManager(configs map[string]ClientConfig, logger *zap.Logger) (*ClientManager, error) {
	m := &ClientManager{
		clients: make(map[string]*StreamingGRPCClient),
		configs: configs,
		logger:  logger,
	}

	// 初始化所有配置的客户端
	for name, cfg := range configs {
		client, err := NewStreamingGRPCClient(cfg, logger.With(zap.String("service", name)))
		if err != nil {
			// 关闭已创建的客户端
			m.Close()
			return nil, fmt.Errorf("failed to create client for %s: %w", name, err)
		}
		m.clients[name] = client
		logger.Info("initialized grpc service client",
			zap.String("service", name),
			zap.String("address", cfg.Address),
		)
	}

	return m, nil
}

// GetClient 获取指定服务的客户端
func (m *ClientManager) GetClient(service string) (*StreamingGRPCClient, error) {
	m.mu.RLock()
	defer m.mu.RUnlock()

	client, ok := m.clients[service]
	if !ok {
		return nil, fmt.Errorf("service %s not found", service)
	}
	return client, nil
}

// HasService 检查服务是否存在
func (m *ClientManager) HasService(service string) bool {
	m.mu.RLock()
	defer m.mu.RUnlock()
	_, ok := m.clients[service]
	return ok
}

// GetServiceConfig 获取服务配置
func (m *ClientManager) GetServiceConfig(service string) (ClientConfig, bool) {
	m.mu.RLock()
	defer m.mu.RUnlock()
	cfg, ok := m.configs[service]
	return cfg, ok
}

// Services 返回所有注册的服务名
func (m *ClientManager) Services() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	services := make([]string, 0, len(m.clients))
	for name := range m.clients {
		services = append(services, name)
	}
	return services
}

// HealthyServices 返回健康的服务列表
func (m *ClientManager) HealthyServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	healthy := make([]string, 0)
	for name, client := range m.clients {
		if client.IsHealthy() {
			healthy = append(healthy, name)
		}
	}
	return healthy
}

// UnhealthyServices 返回不健康的服务列表
func (m *ClientManager) UnhealthyServices() []string {
	m.mu.RLock()
	defer m.mu.RUnlock()

	unhealthy := make([]string, 0)
	for name, client := range m.clients {
		if !client.IsHealthy() {
			unhealthy = append(unhealthy, name)
		}
	}
	return unhealthy
}

// ServiceHealth 返回服务健康状态摘要
type ServiceHealth struct {
	Name    string
	Address string
	Healthy bool
}

// GetHealthStatus 获取所有服务的健康状态
func (m *ClientManager) GetHealthStatus() []ServiceHealth {
	m.mu.RLock()
	defer m.mu.RUnlock()

	status := make([]ServiceHealth, 0, len(m.clients))
	for name, client := range m.clients {
		status = append(status, ServiceHealth{
			Name:    name,
			Address: client.Address(),
			Healthy: client.IsHealthy(),
		})
	}
	return status
}

// AddClient 动态添加客户端
func (m *ClientManager) AddClient(name string, config ClientConfig) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	if _, exists := m.clients[name]; exists {
		return fmt.Errorf("client %s already exists", name)
	}

	client, err := NewStreamingGRPCClient(config, m.logger.With(zap.String("service", name)))
	if err != nil {
		return fmt.Errorf("failed to create client: %w", err)
	}

	m.clients[name] = client
	m.configs[name] = config

	m.logger.Info("added grpc service client",
		zap.String("service", name),
		zap.String("address", config.Address),
	)

	return nil
}

// RemoveClient 移除客户端
func (m *ClientManager) RemoveClient(name string) error {
	m.mu.Lock()
	defer m.mu.Unlock()

	client, exists := m.clients[name]
	if !exists {
		return fmt.Errorf("client %s not found", name)
	}

	if err := client.Close(); err != nil {
		m.logger.Error("failed to close client",
			zap.String("service", name),
			zap.Error(err),
		)
	}

	delete(m.clients, name)
	delete(m.configs, name)

	m.logger.Info("removed grpc service client",
		zap.String("service", name),
	)

	return nil
}

// Close 关闭所有客户端连接
func (m *ClientManager) Close() {
	m.mu.Lock()
	defer m.mu.Unlock()

	for name, client := range m.clients {
		if err := client.Close(); err != nil {
			m.logger.Error("failed to close client",
				zap.String("service", name),
				zap.Error(err),
			)
		}
	}

	m.clients = make(map[string]*StreamingGRPCClient)
	m.logger.Info("closed all grpc service clients")
}
