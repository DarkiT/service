package main

import (
	"context"
	"fmt"
	"time"

	"github.com/darkit/service"
)

// DatabaseService 数据库服务
type DatabaseService struct {
	*service.BaseService
	config *DatabaseConfig
	db     *Database // 模拟数据库连接
}

type DatabaseConfig struct {
	DSN            string
	MaxConnections int
	ConnectTimeout time.Duration
	RetryAttempts  int
}

type Database struct {
	connected bool
}

// NewDatabaseService 创建数据库服务
func NewDatabaseService(config *DatabaseConfig) *DatabaseService {
	s := &DatabaseService{
		// 数据库服务优先级高，因为其他服务可能依赖它
		BaseService: service.NewBaseService(
			"database",
			nil, // 无依赖
			service.WithPriority(service.PriorityHigh),
		),
		config: config,
	}

	// 设置生命周期钩子
	s.SetInitFunc(s.init)
	s.SetStartFunc(s.start)
	s.SetStopFunc(s.stop)
	s.SetUpdateFunc(s.update)

	return s
}

func (s *DatabaseService) init(ctx context.Context) error {
	// 初始化数据库连接池
	s.db = &Database{}
	return nil
}

func (s *DatabaseService) start(ctx context.Context) error {
	// 尝试建立数据库连接
	for attempt := 1; attempt <= s.config.RetryAttempts; attempt++ {
		select {
		case <-ctx.Done():
			return ctx.Err()
		default:
			if err := s.connect(ctx); err != nil {
				if attempt == s.config.RetryAttempts {
					return fmt.Errorf("failed to connect to database after %d attempts: %w",
						attempt, err)
				}
				time.Sleep(time.Second * time.Duration(attempt))
				continue
			}
			return nil
		}
	}
	return nil
}

func (s *DatabaseService) stop(ctx context.Context) error {
	// 优雅关闭数据库连接
	timeout := time.After(5 * time.Second)
	done := make(chan error)

	go func() {
		done <- s.disconnect()
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timeout:
		return fmt.Errorf("timeout while closing database connection")
	case err := <-done:
		return err
	}
}

func (s *DatabaseService) update(ctx context.Context, cfg interface{}) error {
	// 类型安全的配置更新
	dbConfig, ok := cfg.(*DatabaseConfig)
	if !ok {
		return fmt.Errorf("invalid config type: %T", cfg)
	}

	// 验证新配置
	if err := s.validateConfig(dbConfig); err != nil {
		return err
	}

	s.config = dbConfig
	return nil
}

func (s *DatabaseService) connect(ctx context.Context) error {
	// 模拟数据库连接
	timer := time.NewTimer(s.config.ConnectTimeout)
	defer timer.Stop()

	connChan := make(chan error)
	go func() {
		// 模拟连接过程
		time.Sleep(100 * time.Millisecond)
		s.db.connected = true
		connChan <- nil
	}()

	select {
	case <-ctx.Done():
		return ctx.Err()
	case <-timer.C:
		return fmt.Errorf("connection timeout")
	case err := <-connChan:
		return err
	}
}

func (s *DatabaseService) disconnect() error {
	// 模拟断开连接
	s.db.connected = false
	return nil
}

func (s *DatabaseService) validateConfig(cfg *DatabaseConfig) error {
	if cfg.DSN == "" {
		return fmt.Errorf("DSN cannot be empty")
	}
	if cfg.MaxConnections <= 0 {
		return fmt.Errorf("MaxConnections must be positive")
	}
	if cfg.ConnectTimeout <= 0 {
		return fmt.Errorf("ConnectTimeout must be positive")
	}
	if cfg.RetryAttempts <= 0 {
		return fmt.Errorf("RetryAttempts must be positive")
	}
	return nil
}

// HealthCheck 实现健康检查
func (s *DatabaseService) HealthCheck(ctx context.Context) error {
	if !s.db.connected {
		return &service.ServiceError{
			Code:    service.ErrInvalidState,
			Message: "database is not connected",
		}
	}
	return nil
}
