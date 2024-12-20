package main

import (
	"context"
	"log"
	"time"

	"github.com/darkit/service"
)

func main() {
	// 创建上下文
	ctx := context.Background()

	// 创建服务组
	sg := service.NewServiceGroup(ctx, service.ServiceGroupOptions{
		StartTimeout:        time.Minute,
		StopTimeout:         time.Minute,
		HealthCheckInterval: time.Second * 30,
	})

	// 创建数据库服务
	dbService := NewDatabaseService(&DatabaseConfig{
		DSN:            "postgres://localhost:5432/mydb",
		MaxConnections: 10,
		ConnectTimeout: time.Second * 5,
		RetryAttempts:  3,
	})

	// 创建 API 服务
	apiService := NewAPIService(&APIConfig{
		Port:         8080,
		ReadTimeout:  time.Second * 30,
		WriteTimeout: time.Second * 30,
	}, dbService)

	// 添加事件监听
	sg.AddEventListener(service.EventStateChange, &service.DefaultEventListener{
		OnEventFunc: func(event service.ServiceEvent) {
			log.Printf("Service %s state changed to %s",
				event.ServiceName, event.State)
		},
	})

	// 添加服务到服务组
	if err := sg.Add(dbService); err != nil {
		log.Fatalf("Failed to add database service: %v", err)
	}
	if err := sg.Add(apiService); err != nil {
		log.Fatalf("Failed to add API service: %v", err)
	}

	// 启动服务组
	if err := sg.Start(); err != nil {
		log.Fatalf("Failed to start services: %v", err)
	}

	// 等待服务启动完成
	if err := sg.WaitForStart(ctx); err != nil {
		log.Fatalf("Error waiting for services to start: %v", err)
	}

	// 获取服务状态
	state := sg.GetGroupState()
	log.Printf("Running services: %d/%d", state.RunningServices, state.TotalServices)

	// 监控服务指标
	go func() {
		ticker := time.NewTicker(time.Minute)
		defer ticker.Stop()

		for {
			select {
			case <-ctx.Done():
				return
			case <-ticker.C:
				if metrics, err := sg.GetServiceMetrics("database"); err == nil {
					log.Printf("Database service metrics: %+v", metrics)
				}
				if metrics, err := sg.GetServiceMetrics("api"); err == nil {
					log.Printf("API service metrics: %+v", metrics)
				}
			}
		}
	}()

	// 等待中断信号
	<-ctx.Done()

	// 优雅停止服务
	stopCtx, cancel := context.WithTimeout(context.Background(), time.Minute)
	defer cancel()

	if err := sg.GracefulStop(stopCtx); err != nil {
		log.Printf("Error stopping services: %v", err)
	}
}
