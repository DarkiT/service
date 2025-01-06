package service

import (
	"context"
	"fmt"
	"sync"
	"sync/atomic"
	"time"
)

// ServiceGroup 管理一组服务
type ServiceGroup struct {
	services sync.Map
	depGraph *DependencyGraph
	ctx      context.Context
	cancel   context.CancelFunc

	// 配置选项
	options ServiceGroupOptions

	// 状态追踪
	startupWg  sync.WaitGroup
	shutdownWg sync.WaitGroup
	startupErr error
	isStarting atomic.Bool

	metrics *MetricsCollector
	events  *EventManager
}

// ServiceGroupOptions 配置选项
type ServiceGroupOptions struct {
	StartTimeout        time.Duration
	StopTimeout         time.Duration
	HealthCheckInterval time.Duration
}

// DefaultServiceGroupOptions 默认配置
var DefaultServiceGroupOptions = ServiceGroupOptions{
	StartTimeout:        time.Minute,
	StopTimeout:         time.Minute,
	HealthCheckInterval: time.Second * 30,
}

// NewServiceGroup 创建新的服务组
func NewServiceGroup(ctx context.Context, opts ...ServiceGroupOptions) *ServiceGroup {
	options := DefaultServiceGroupOptions
	if len(opts) > 0 {
		options = opts[0]
		// 确保健康检查间隔有效
		if options.HealthCheckInterval <= 0 {
			options.HealthCheckInterval = DefaultServiceGroupOptions.HealthCheckInterval
		}
	}

	ctx, cancel := context.WithCancel(ctx)
	sg := &ServiceGroup{
		depGraph: NewDependencyGraph(),
		ctx:      ctx,
		cancel:   cancel,
		options:  options,
		metrics:  NewMetricsCollector(),
		events:   NewEventManager(),
	}
	return sg
}

// Add 添加服务到组
func (sg *ServiceGroup) Add(s Service) error {
	if _, loaded := sg.services.LoadOrStore(s.Name(), s); loaded {
		return &ServiceError{
			Code:    ErrServiceAlreadyExists,
			Message: fmt.Sprintf("service %s already exists", s.Name()),
		}
	}

	// 创建服务节点
	node := &ServiceNode{
		Name:     s.Name(),
		Priority: s.Priority(),
		Deps:     s.Dependencies(),
	}

	// 添加到依赖图
	if err := sg.depGraph.AddNode(node); err != nil {
		sg.services.Delete(s.Name())
		return err
	}

	// 注册服务指标
	sg.metrics.RegisterService(s.Name())

	defaultLogger.Info("Added service to ServiceGroup",
		"service", s.Name(),
		"priority", s.Priority(),
		"dependencies", s.Dependencies())
	return nil
}

// Start 启动所有服务
func (sg *ServiceGroup) Start() error {
	if !sg.isStarting.CompareAndSwap(false, true) {
		return &ServiceError{
			Code:    ErrInvalidState,
			Message: "services are already starting",
		}
	}

	// 获取启动顺序
	startOrder, err := sg.depGraph.GetStartOrder()
	if err != nil {
		return err
	}

	// 创建启动上下文
	ctx, cancel := context.WithTimeout(sg.ctx, sg.options.StartTimeout)
	defer cancel()

	// 按顺序启动服务
	for _, name := range startOrder {
		if err := sg.startService(ctx, name); err != nil {
			return err
		}
	}

	// 启动健康检查（如果间隔大于0）
	if sg.options.HealthCheckInterval > 0 {
		go sg.healthCheckLoop()
	}

	return nil
}

// Stop 停止所有服务
func (sg *ServiceGroup) Stop() error {
	sg.cancel() // 触发所有服务停止

	// 创建停止上下文
	ctx, cancel := context.WithTimeout(context.Background(), sg.options.StopTimeout)
	defer cancel()

	// 获取逆序的启动顺序作为停止顺序
	stopOrder, err := sg.depGraph.GetStartOrder()
	if err != nil {
		return err
	}
	for i := len(stopOrder)/2 - 1; i >= 0; i-- {
		opp := len(stopOrder) - 1 - i
		stopOrder[i], stopOrder[opp] = stopOrder[opp], stopOrder[i]
	}

	// 按顺序停止服务
	var stopErr error
	for _, name := range stopOrder {
		if err := sg.stopService(ctx, name); err != nil {
			stopErr = err
			defaultLogger.Error("Error stopping service",
				"service", name,
				"error", err)
		}
	}

	return stopErr
}

// startService 启动单个服务
func (sg *ServiceGroup) startService(ctx context.Context, name string) error {
	service, ok := sg.services.Load(name)
	if !ok {
		return &ServiceError{
			Code:    ErrServiceNotFound,
			Message: fmt.Sprintf("service %s not found", name),
		}
	}

	s := service.(Service)
	if err := s.Start(ctx); err != nil {
		return &ServiceError{
			Code:    ErrStartupFailed,
			Message: fmt.Sprintf("failed to start service %s", name),
			Err:     err,
		}
	}

	return nil
}

// stopService 停止单个服务
func (sg *ServiceGroup) stopService(ctx context.Context, name string) error {
	svc, ok := sg.services.Load(name)
	if !ok {
		return &ServiceError{
			Code:    ErrServiceNotFound,
			Message: fmt.Sprintf("service %s not found", name),
		}
	}

	service := svc.(Service)

	// 记录停止指标
	sg.metrics.RecordStop(name)

	if err := service.Stop(ctx); err != nil {
		// 记录错误指标
		sg.metrics.RecordError(name, err)
		return fmt.Errorf("failed to stop service %s: %w", name, err)
	}

	return nil
}

// healthCheckLoop 运行健康检查循环
func (sg *ServiceGroup) healthCheckLoop() {
	ticker := time.NewTicker(sg.options.HealthCheckInterval)
	defer ticker.Stop()

	for {
		select {
		case <-sg.ctx.Done():
			return
		case <-ticker.C:
			sg.services.Range(func(key, value interface{}) bool {
				service := value.(Service)
				if err := service.HealthCheck(sg.ctx); err != nil {
					defaultLogger.Error("Service health check failed",
						"service", service.Name(),
						"error", err)
				}
				return true
			})
		}
	}
}

// WaitForStart 等待所有服务启动完成
func (sg *ServiceGroup) WaitForStart(ctx context.Context) error {
	done := make(chan struct{})
	go func() {
		sg.startupWg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return &ServiceError{
			Code:    ErrStartupTimeout,
			Message: "timeout waiting for services to start",
			Err:     ctx.Err(),
		}
	case <-done:
		return sg.startupErr
	}
}

// GracefulStop 优雅停止所有服务
func (sg *ServiceGroup) GracefulStop(ctx context.Context) error {
	// 先停止健康检查
	sg.cancel()

	// 创建一个新的 context 用于停止操作
	stopCtx, cancel := context.WithTimeout(ctx, sg.options.StopTimeout)
	defer cancel()

	// 获取停止顺序（依赖关系的反序）
	stopOrder, err := sg.depGraph.GetStartOrder()
	if err != nil {
		return &ServiceError{
			Code:    ErrShutdownFailed,
			Message: "failed to determine service stop order",
			Err:     err,
		}
	}

	// 反转顺序
	for i := len(stopOrder)/2 - 1; i >= 0; i-- {
		opp := len(stopOrder) - 1 - i
		stopOrder[i], stopOrder[opp] = stopOrder[opp], stopOrder[i]
	}

	// 跟踪停止进度
	sg.shutdownWg.Add(len(stopOrder))

	// 按顺序停止服务
	for _, name := range stopOrder {
		go func(serviceName string) {
			defer sg.shutdownWg.Done()

			if err := sg.stopService(stopCtx, serviceName); err != nil {
				sg.events.PublishEvent(ServiceEvent{
					ServiceName: serviceName,
					EventType:   EventStop,
					Error:       err,
					Time:        time.Now(),
				})
			}
		}(name)
	}

	// 等待所有服务停止或超时
	done := make(chan struct{})
	go func() {
		sg.shutdownWg.Wait()
		close(done)
	}()

	select {
	case <-ctx.Done():
		return &ServiceError{
			Code:    ErrShutdownTimeout,
			Message: "timeout waiting for services to stop",
			Err:     ctx.Err(),
		}
	case <-done:
		return nil
	}
}

// ServiceGroupState 服务组状态
type ServiceGroupState struct {
	TotalServices   int
	RunningServices int
	FailedServices  int
	ServiceStates   map[string]ServiceState
}

// GetGroupState 获取服务组状态
func (sg *ServiceGroup) GetGroupState() ServiceGroupState {
	state := ServiceGroupState{
		ServiceStates: make(map[string]ServiceState),
	}

	sg.services.Range(func(key, value interface{}) bool {
		service := value.(Service)
		serviceState := service.State()
		state.TotalServices++
		state.ServiceStates[key.(string)] = serviceState

		switch serviceState {
		case StateRunning:
			state.RunningServices++
		case StateError:
			state.FailedServices++
		}
		return true
	})

	return state
}

// GetServiceMetrics 获取服务指标
func (sg *ServiceGroup) GetServiceMetrics(name string) (*ServiceMetrics, error) {
	metrics, exists := sg.metrics.GetMetrics(name)
	if !exists {
		return nil, &ServiceError{
			Code:    ErrServiceNotFound,
			Message: fmt.Sprintf("service %s not found", name),
		}
	}
	return metrics, nil
}

// AddEventListener 添加事件监听器
func (sg *ServiceGroup) AddEventListener(eventType EventType, listener EventListener) {
	sg.events.AddListener(eventType, listener)
}
