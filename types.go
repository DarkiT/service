package service

import (
	"context"
)

// ServiceState 定义服务状态
type ServiceState int32

const (
	StateUninitialized ServiceState = iota
	StateInitialized
	StateStarting
	StateRunning
	StateStopping
	StateStopped
	StateError
)

// String 实现 Stringer 接口
func (s ServiceState) String() string {
	return [...]string{
		"Uninitialized",
		"Initialized",
		"Starting",
		"Running",
		"Stopping",
		"Stopped",
		"Error",
	}[s]
}

// ServiceError 定义统一的错误类型
type ServiceError struct {
	Code    ErrorCode
	Message string
	Err     error
}

// ErrorCode 定义错误码
type ErrorCode int

const (
	ErrNone ErrorCode = iota
	ErrServiceNotFound
	ErrServiceAlreadyExists
	ErrInvalidState
	ErrStartupTimeout
	ErrStartupFailed
	ErrShutdownTimeout
	ErrShutdownFailed
	ErrDependencyFailed
)

// Error 实现 error 接口
func (e *ServiceError) Error() string {
	if e.Err != nil {
		return e.Message + ": " + e.Err.Error()
	}
	return e.Message
}

// ErrorCode 的字符串表示
func (e ErrorCode) String() string {
	return [...]string{
		"None",
		"ServiceNotFound",
		"ServiceAlreadyExists",
		"InvalidState",
		"StartupTimeout",
		"StartupFailed",
		"ShutdownTimeout",
		"ShutdownFailed",
		"DependencyFailed",
	}[e]
}

// ServicePriority 服务优先级
type ServicePriority int

const (
	PriorityHighest ServicePriority = 0
	PriorityHigh    ServicePriority = 25
	PriorityNormal  ServicePriority = 50
	PriorityLow     ServicePriority = 75
	PriorityLowest  ServicePriority = 100
)

// Service 定义服务接口
type Service interface {
	// 基础生命周期方法

	Init(ctx context.Context) error
	Start(ctx context.Context) error
	Stop(ctx context.Context) error

	// 服务信息

	Name() string
	State() ServiceState

	// HealthCheck 健康检查
	HealthCheck(ctx context.Context) error

	// Update 配置更新
	Update(ctx context.Context, config interface{}) error

	// Dependencies 依赖管理
	Dependencies() []string

	// Priority 返回服务优先级
	Priority() ServicePriority
}
