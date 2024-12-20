package service

import (
	"fmt"
	"sync/atomic"
)

// StateMachine 实现服务状态管理
type StateMachine struct {
	state        atomic.Int32
	transitions  map[ServiceState][]ServiceState
	onTransition func(from, to ServiceState)
}

// NewStateMachine 创建新的状态机
func NewStateMachine(initial ServiceState, onTransition func(from, to ServiceState)) *StateMachine {
	sm := &StateMachine{
		transitions:  makeDefaultTransitions(),
		onTransition: onTransition,
	}
	sm.state.Store(int32(initial))
	return sm
}

// makeDefaultTransitions 创建默认的状态转换规则
func makeDefaultTransitions() map[ServiceState][]ServiceState {
	return map[ServiceState][]ServiceState{
		StateUninitialized: {StateInitialized},
		StateInitialized:   {StateStarting},
		StateStarting:      {StateRunning, StateError},
		StateRunning:       {StateStopping, StateError},
		StateStopping:      {StateStopped, StateError},
		StateStopped:       {StateStarting},
		StateError:         {StateInitialized, StateStopped},
	}
}

// TransitionTo 尝试转换到新状态
func (sm *StateMachine) TransitionTo(newState ServiceState) error {
	currentState := ServiceState(sm.state.Load())

	// 检查状态转换是否合法
	if !sm.isValidTransition(currentState, newState) {
		return &ServiceError{
			Code:    ErrInvalidState,
			Message: fmt.Sprintf("invalid state transition from %s to %s", currentState, newState),
		}
	}

	// 尝试更新状态
	if !sm.state.CompareAndSwap(int32(currentState), int32(newState)) {
		return &ServiceError{
			Code:    ErrInvalidState,
			Message: "state was changed by another goroutine",
		}
	}

	// 调用状态转换回调
	if sm.onTransition != nil {
		sm.onTransition(currentState, newState)
	}

	return nil
}

// isValidTransition 检查状态转换是否合法
func (sm *StateMachine) isValidTransition(from, to ServiceState) bool {
	allowed, exists := sm.transitions[from]
	if !exists {
		return false
	}
	for _, state := range allowed {
		if state == to {
			return true
		}
	}
	return false
}

// Current 获取当前状态
func (sm *StateMachine) Current() ServiceState {
	return ServiceState(sm.state.Load())
}

// AddTransition 添加自定义状态转换规则
func (sm *StateMachine) AddTransition(from ServiceState, to ...ServiceState) {
	if _, exists := sm.transitions[from]; !exists {
		sm.transitions[from] = make([]ServiceState, 0)
	}
	sm.transitions[from] = append(sm.transitions[from], to...)
}

// RemoveTransition 移除状态转换规则
func (sm *StateMachine) RemoveTransition(from, to ServiceState) {
	if allowed, exists := sm.transitions[from]; exists {
		for i, state := range allowed {
			if state == to {
				sm.transitions[from] = append(allowed[:i], allowed[i+1:]...)
				break
			}
		}
	}
}

// Reset 重置状态机到初始状态
func (sm *StateMachine) Reset(initialState ServiceState) {
	sm.state.Store(int32(initialState))
}
