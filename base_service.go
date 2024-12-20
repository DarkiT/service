package service

import (
	"context"
	"fmt"
	"sync"
	"time"
)

// BaseService 提供基础服务实现
type BaseService struct {
	name         string
	deps         []string
	priority     ServicePriority
	stateMachine *StateMachine

	// 生命周期回调
	initFunc   func(context.Context) error
	startFunc  func(context.Context) error
	stopFunc   func(context.Context) error
	updateFunc func(context.Context, interface{}) error

	mu     sync.RWMutex
	ctx    context.Context
	cancel context.CancelFunc
}

// SetInitFunc 设置初始化回调
func (bs *BaseService) SetInitFunc(f func(context.Context) error) {
	bs.initFunc = f
}

// SetStartFunc 设置启动回调
func (bs *BaseService) SetStartFunc(f func(context.Context) error) {
	bs.startFunc = f
}

// SetStopFunc 设置停止回调
func (bs *BaseService) SetStopFunc(f func(context.Context) error) {
	bs.stopFunc = f
}

// SetUpdateFunc 设置更新回调
func (bs *BaseService) SetUpdateFunc(f func(context.Context, interface{}) error) {
	bs.updateFunc = f
}

// NewBaseService 创建新的基础服务
func NewBaseService(name string, deps []string, opts ...ServiceOption) *BaseService {
	bs := &BaseService{
		name:     name,
		deps:     deps,
		priority: PriorityNormal, // 默认优先级
	}

	// 应用选项
	for _, opt := range opts {
		opt(bs)
	}

	bs.stateMachine = NewStateMachine(StateUninitialized, bs.handleStateChange)
	return bs
}

// handleStateChange 处理状态变更
func (bs *BaseService) handleStateChange(from, to ServiceState) {
	// 这里可以添加日志记录或监控
}

// 实现 Service 接口
func (bs *BaseService) Name() string {
	return bs.name
}

func (bs *BaseService) State() ServiceState {
	return bs.stateMachine.Current()
}

func (bs *BaseService) Dependencies() []string {
	return bs.deps
}

// Init 初始化服务
func (bs *BaseService) Init(ctx context.Context) error {
	// 先转换到初始化状态
	if err := bs.stateMachine.TransitionTo(StateInitialized); err != nil {
		return fmt.Errorf("failed to transition to Initialized state: %w", err)
	}

	// 执行初始化回调
	if bs.initFunc != nil {
		if err := bs.initFunc(ctx); err != nil {
			bs.stateMachine.TransitionTo(StateError)
			return fmt.Errorf("init function failed: %w", err)
		}
	}

	return nil
}

// Start 启动服务
func (bs *BaseService) Start(ctx context.Context) error {
	// 先初始化
	if bs.State() == StateUninitialized {
		if err := bs.Init(ctx); err != nil {
			return fmt.Errorf("failed to initialize service: %w", err)
		}
	}

	// 然后转换到 Starting 状态
	if err := bs.stateMachine.TransitionTo(StateStarting); err != nil {
		return fmt.Errorf("failed to transition to Starting state: %w", err)
	}

	// 执行启动回调
	if bs.startFunc != nil {
		if err := bs.startFunc(ctx); err != nil {
			bs.stateMachine.TransitionTo(StateError)
			return fmt.Errorf("start function failed: %w", err)
		}
	}

	// 最后转换到 Running 状态
	return bs.stateMachine.TransitionTo(StateRunning)
}

// Stop 停止服务
func (bs *BaseService) Stop(ctx context.Context) error {
	if err := bs.stateMachine.TransitionTo(StateStopping); err != nil {
		return err
	}

	if bs.stopFunc != nil {
		if err := bs.stopFunc(ctx); err != nil {
			bs.stateMachine.TransitionTo(StateError)
			return err
		}
	}

	return bs.stateMachine.TransitionTo(StateStopped)
}

// Update 更新服务配置
func (bs *BaseService) Update(ctx context.Context, config interface{}) error {
	if bs.updateFunc != nil {
		return bs.updateFunc(ctx, config)
	}
	return nil
}

func (bs *BaseService) HealthCheck(ctx context.Context) error {
	if bs.State() != StateRunning {
		return &ServiceError{
			Code:    ErrInvalidState,
			Message: "service is not running",
		}
	}
	return nil
}

// RetryOptions 重试选项
type RetryOptions struct {
	MaxAttempts int
	Delay       time.Duration
	MaxDelay    time.Duration
}

// StartWithRetry 带重试的服务启动
func (bs *BaseService) StartWithRetry(ctx context.Context, opts RetryOptions) error {
	var lastErr error
	attempt := 0
	delay := opts.Delay

	for attempt < opts.MaxAttempts {
		if err := bs.Start(ctx); err != nil {
			lastErr = err
			attempt++

			if attempt >= opts.MaxAttempts {
				break
			}

			select {
			case <-ctx.Done():
				return ctx.Err()
			case <-time.After(delay):
				delay = min(delay*2, opts.MaxDelay)
				continue
			}
		}
		return nil
	}

	return fmt.Errorf("service start failed after %d attempts: %w", attempt, lastErr)
}

// ServiceOption 服务配置选项
type ServiceOption func(*BaseService)

// WithPriority 设置服务优先级
func WithPriority(priority ServicePriority) ServiceOption {
	return func(bs *BaseService) {
		bs.priority = priority
	}
}

// Priority 实现 Service 接口
func (bs *BaseService) Priority() ServicePriority {
	return bs.priority
}
