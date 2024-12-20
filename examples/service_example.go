package main

import (
	"context"
	"fmt"
	"time"

	"github.com/darkit/service"
)

// ExampleService 示例服务实现
type ExampleService struct {
	*service.BaseService
	data string
}

// NewExampleService 创建示例服务
func NewExampleService(name string, deps []string) *ExampleService {
	s := &ExampleService{
		BaseService: service.NewBaseService(name, deps),
	}

	// 设置生命周期钩子
	s.SetInitFunc(s.init)
	s.SetStartFunc(s.start)
	s.SetStopFunc(s.stop)

	return s
}

func (s *ExampleService) init(ctx context.Context) error {
	fmt.Printf("[%s] Initializing...\n", s.Name())
	s.data = "initialized"
	return nil
}

func (s *ExampleService) start(ctx context.Context) error {
	fmt.Printf("[%s] Starting...\n", s.Name())
	s.data = "running"
	return nil
}

func (s *ExampleService) stop(ctx context.Context) error {
	fmt.Printf("[%s] Stopping...\n", s.Name())
	s.data = "stopped"
	return nil
}

// 主函数示例
func main() {
	ctx := context.Background()

	// 创建服务组
	sg := service.NewServiceGroup(ctx, service.ServiceGroupOptions{
		StartTimeout: time.Minute,
		StopTimeout:  time.Minute,
	})

	// 创建服务
	service1 := NewExampleService("service1", nil)
	service2 := NewExampleService("service2", []string{"service1"})
	service3 := NewExampleService("service3", []string{"service2"})

	// 添加事件监听
	sg.AddEventListener(service.EventStateChange, &service.DefaultEventListener{
		OnEventFunc: func(event service.ServiceEvent) {
			fmt.Printf("Service %s state changed to %s\n",
				event.ServiceName, event.State)
		},
	})

	// 添加服务到服务组
	sg.Add(service1)
	sg.Add(service2)
	sg.Add(service3)

	// 启动服务组
	if err := sg.Start(); err != nil {
		fmt.Printf("Failed to start services: %v\n", err)
		return
	}

	// 等待服务启动完成
	if err := sg.WaitForStart(ctx); err != nil {
		fmt.Printf("Error waiting for services to start: %v\n", err)
		return
	}

	// 获取服务状态
	state := sg.GetGroupState()
	fmt.Printf("Running services: %d/%d\n",
		state.RunningServices, state.TotalServices)

	// 获取服务指标
	if metrics, err := sg.GetServiceMetrics("service1"); err == nil {
		fmt.Printf("Service1 uptime: %v\n", metrics.TotalUptime)
	}

	// 优雅停止服务
	if err := sg.GracefulStop(ctx); err != nil {
		fmt.Printf("Error stopping services: %v\n", err)
		return
	}
}
