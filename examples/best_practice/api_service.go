package main

import (
	"context"
	"net/http"
	"time"

	"github.com/darkit/service"
)

// APIService 依赖数据库服务的 API 服务
type APIService struct {
	*service.BaseService
	config     *APIConfig
	dbService  *DatabaseService
	httpServer *http.Server
}

type APIConfig struct {
	Port         int
	ReadTimeout  time.Duration
	WriteTimeout time.Duration
}

func NewAPIService(config *APIConfig, dbService *DatabaseService) *APIService {
	s := &APIService{
		BaseService: service.NewBaseService(
			"api",
			[]string{"database"}, // 声明依赖数据库服务
			service.WithPriority(service.PriorityNormal),
		),
		config:    config,
		dbService: dbService,
	}

	// 设置生命周期回调
	s.SetInitFunc(s.init)
	s.SetStartFunc(s.start)
	s.SetStopFunc(s.stop)
	s.SetUpdateFunc(s.update)

	return s
}

func (s *APIService) init(ctx context.Context) error {
	// 初始化逻辑
	return nil
}

func (s *APIService) start(ctx context.Context) error {
	// 启动逻辑
	return nil
}

func (s *APIService) stop(ctx context.Context) error {
	// 停止逻辑
	return nil
}

// update 处理配置更新
func (s *APIService) update(ctx context.Context, cfg interface{}) error {
	// 配置更新
	return nil
}
