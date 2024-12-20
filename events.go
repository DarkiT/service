package service

import (
	"sync"
	"time"
)

// EventType 事件类型
type EventType string

const (
	EventInit        EventType = "Init"
	EventStart       EventType = "Start"
	EventStop        EventType = "Stop"
	EventRestart     EventType = "Restart"
	EventError       EventType = "Error"
	EventHealthCheck EventType = "HealthCheck"
	EventStateChange EventType = "StateChange"
)

// ServiceEvent 服务事件
type ServiceEvent struct {
	ServiceName string
	EventType   EventType
	State       ServiceState
	Time        time.Time
	Error       error
	Metadata    map[string]interface{} // 额外的事件元数据
}

// EventListener 事件监听器接口
type EventListener interface {
	OnServiceEvent(event ServiceEvent)
}

// EventManager 事件管理器
type EventManager struct {
	mu        sync.RWMutex
	listeners map[EventType][]EventListener
}

// NewEventManager 创建新的事件管理器
func NewEventManager() *EventManager {
	return &EventManager{
		listeners: make(map[EventType][]EventListener),
	}
}

// AddListener 添加事件监听器
func (em *EventManager) AddListener(eventType EventType, listener EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if _, exists := em.listeners[eventType]; !exists {
		em.listeners[eventType] = make([]EventListener, 0)
	}
	em.listeners[eventType] = append(em.listeners[eventType], listener)
}

// RemoveListener 移除事件监听器
func (em *EventManager) RemoveListener(eventType EventType, listener EventListener) {
	em.mu.Lock()
	defer em.mu.Unlock()

	if listeners, exists := em.listeners[eventType]; exists {
		for i, l := range listeners {
			if l == listener {
				em.listeners[eventType] = append(listeners[:i], listeners[i+1:]...)
				break
			}
		}
	}
}

// PublishEvent 发布事件
func (em *EventManager) PublishEvent(event ServiceEvent) {
	em.mu.RLock()
	defer em.mu.RUnlock()

	// 通知特定类型的监听器
	if listeners, exists := em.listeners[event.EventType]; exists {
		for _, listener := range listeners {
			go listener.OnServiceEvent(event)
		}
	}

	// 通知通用监听器（如果有的话）
	if listeners, exists := em.listeners["*"]; exists {
		for _, listener := range listeners {
			go listener.OnServiceEvent(event)
		}
	}
}

// DefaultEventListener 默认事件监听器实现
type DefaultEventListener struct {
	OnEventFunc func(event ServiceEvent)
}

func (l *DefaultEventListener) OnServiceEvent(event ServiceEvent) {
	if l.OnEventFunc != nil {
		l.OnEventFunc(event)
	}
}
