package service

import (
	"sync"
	"sync/atomic"
	"time"
)

// ServiceMetrics 服务指标
type ServiceMetrics struct {
	StartTime         time.Time
	RestartCount      atomic.Int64
	LastError         error
	LastErrorTime     time.Time
	State             ServiceState
	HealthCheckCount  atomic.Int64
	HealthCheckErrors atomic.Int64
	LastHealthCheck   time.Time
	TotalUptime       time.Duration
	LastStateChange   time.Time
}

// MetricsCollector 指标收集器
type MetricsCollector struct {
	mu      sync.RWMutex
	metrics map[string]*ServiceMetrics
}

// NewMetricsCollector 创建新的指标收集器
func NewMetricsCollector() *MetricsCollector {
	return &MetricsCollector{
		metrics: make(map[string]*ServiceMetrics),
	}
}

// RegisterService 注册服务到指标收集器
func (mc *MetricsCollector) RegisterService(serviceName string) {
	mc.mu.Lock()
	defer mc.mu.Unlock()

	if _, exists := mc.metrics[serviceName]; !exists {
		mc.metrics[serviceName] = &ServiceMetrics{
			LastStateChange: time.Now(),
		}
	}
}

// RecordStart 记录服务启动
func (mc *MetricsCollector) RecordStart(serviceName string) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[serviceName]
	mc.mu.RUnlock()

	if exists {
		metrics.StartTime = time.Now()
		metrics.State = StateRunning
		metrics.LastStateChange = time.Now()
	}
}

// RecordStop 记录服务停止
func (mc *MetricsCollector) RecordStop(serviceName string) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[serviceName]
	mc.mu.RUnlock()

	if exists {
		metrics.State = StateStopped
		metrics.LastStateChange = time.Now()
		metrics.TotalUptime += time.Since(metrics.StartTime)
	}
}

// RecordRestart 记录服务重启
func (mc *MetricsCollector) RecordRestart(serviceName string) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[serviceName]
	mc.mu.RUnlock()

	if exists {
		metrics.RestartCount.Add(1)
		metrics.StartTime = time.Now()
		metrics.LastStateChange = time.Now()
	}
}

// RecordError 记录服务错误
func (mc *MetricsCollector) RecordError(serviceName string, err error) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[serviceName]
	mc.mu.RUnlock()

	if exists {
		metrics.LastError = err
		metrics.LastErrorTime = time.Now()
		metrics.State = StateError
		metrics.LastStateChange = time.Now()
	}
}

// RecordHealthCheck 记录健康检查
func (mc *MetricsCollector) RecordHealthCheck(serviceName string, err error) {
	mc.mu.RLock()
	metrics, exists := mc.metrics[serviceName]
	mc.mu.RUnlock()

	if exists {
		metrics.HealthCheckCount.Add(1)
		metrics.LastHealthCheck = time.Now()
		if err != nil {
			metrics.HealthCheckErrors.Add(1)
		}
	}
}

// GetMetrics 获取服务指标
func (mc *MetricsCollector) GetMetrics(serviceName string) (*ServiceMetrics, bool) {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	metrics, exists := mc.metrics[serviceName]
	if !exists {
		return nil, false
	}

	// 返回指标的副本
	metricsCopy := *metrics
	return &metricsCopy, true
}

// GetAllMetrics 获取所有服务的指标
func (mc *MetricsCollector) GetAllMetrics() map[string]ServiceMetrics {
	mc.mu.RLock()
	defer mc.mu.RUnlock()

	result := make(map[string]ServiceMetrics)
	for name, metrics := range mc.metrics {
		result[name] = *metrics
	}
	return result
}
