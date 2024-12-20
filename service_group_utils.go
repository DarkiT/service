package service

import (
	"context"
	"fmt"
)

// GetService 获取指定服务
func (sg *ServiceGroup) GetService(name string) (Service, error) {
	if svc, ok := sg.services.Load(name); ok {
		return svc.(Service), nil
	}
	return nil, &ServiceError{
		Code:    ErrServiceNotFound,
		Message: fmt.Sprintf("service %s not found", name),
	}
}

// UpdateService 更新服务配置
func (sg *ServiceGroup) UpdateService(ctx context.Context, name string, config interface{}) error {
	svc, err := sg.GetService(name)
	if err != nil {
		return err
	}
	return svc.Update(ctx, config)
}

// RestartService 重启指定服务
func (sg *ServiceGroup) RestartService(ctx context.Context, name string) error {
	_, err := sg.GetService(name)
	if err != nil {
		return err
	}

	if err := sg.stopService(ctx, name); err != nil {
		return err
	}

	return sg.startService(ctx, name)
}

// ListServices 列出所有服务
func (sg *ServiceGroup) ListServices() []string {
	var services []string
	sg.services.Range(func(key, _ interface{}) bool {
		services = append(services, key.(string))
		return true
	})
	return services
}

// GetServiceStates 获取所有服务状态
func (sg *ServiceGroup) GetServiceStates() map[string]ServiceState {
	states := make(map[string]ServiceState)
	sg.services.Range(func(key, value interface{}) bool {
		service := value.(Service)
		states[key.(string)] = service.State()
		return true
	})
	return states
}
