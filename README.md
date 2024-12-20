# Service 服务管理框架

[![Go Reference](https://pkg.go.dev/badge/github.com/darkit/service.svg)](https://pkg.go.dev/github.com/darkit/service)
[![Go Report Card](https://goreportcard.com/badge/github.com/darkit/service)](https://goreportcard.com/report/github.com/darkit/service)
[![MIT License](https://img.shields.io/badge/license-MIT-blue.svg)](https://github.com/darkit/service/blob/master/LICENSE)

service 提供了一个用于管理服务生命周期的轻量级框架，支持服务依赖管理、状态监控、指标收集等功能。

## 特性

- 服务生命周期管理
  - 完整的状态机实现
  - 支持优雅启动和关闭
  - 自动管理服务依赖关系

- 状态管理
  - 完整的状态机实现
  - 状态转换验证
  - 状态变更通知

- 依赖管理
  - 自动解析服务依赖关系
  - 检测循环依赖
  - 按依赖顺序启动和停止
  - 支持服务优先级

- 监控和指标
  - 服务健康检查
  - 详细的服务指标
  - 运行时状态监控

- 事件系统
  - 异步事件通知
  - 可自定义事件监听器
  - 支持多种事件类型

## 安装

```bash
go get github.com/darkit/service
```

## 快速开始

1. 创建服务

```go
type MyService struct {
    *service.BaseService
}

// 创建普通优先级服务
func NewMyService(name string) *MyService {
    s := &MyService{
        BaseService: service.NewBaseService(name, nil),
    }
    
    // 设置生命周期回调
    s.SetInitFunc(s.init)
    s.SetStartFunc(s.start)
    s.SetStopFunc(s.stop)
    
    return s
}

// 创建高优先级服务
func NewHighPriorityService(name string) *MyService {
    s := &MyService{
        BaseService: service.NewBaseService(name, nil, service.WithPriority(service.PriorityHigh)),
    }
    return s
}
```

2. 使用服务组

```go
func main() {
    ctx := context.Background()
    
    // 创建服务组
    sg := service.NewServiceGroup(ctx, service.ServiceGroupOptions{
        StartTimeout: time.Minute,
        StopTimeout: time.Minute,
    })
    
    // 添加服务（顺序无关，会根据优先级和依赖关系自动排序）
    sg.Add(NewHighPriorityService("high-priority-service"))
    sg.Add(NewMyService("normal-service"))
    
    // 启动服务
    if err := sg.Start(); err != nil {
        log.Fatal(err)
    }
}
```

## 服务优先级

框架支持五个优先级级别：

```go
const (
    PriorityHighest  ServicePriority = 0
    PriorityHigh     ServicePriority = 25
    PriorityNormal   ServicePriority = 50  // 默认优先级
    PriorityLow      ServicePriority = 75
    PriorityLowest   ServicePriority = 100
)
```

优先级规则：
- 同一依赖层级内，高优先级服务先启动
- 不同依赖层级间，依赖关系优先于优先级
- 未指定优先级时默认为 PriorityNormal

## 服务生命周期

服务状态转换图：

```
Uninitialized -> Initialized -> Starting -> Running  -> Error
Running -> Stopping -> Stopped  -> Error
```

## 配置选项

```go
type ServiceGroupOptions struct {
    StartTimeout        time.Duration // 服务启动超时时间
    StopTimeout         time.Duration // 服务停止超时时间
    HealthCheckInterval time.Duration // 健康检查间隔
}
```

## 服务指标

可用的服务指标：
- 启动时间
- 运行时长
- 重启次数
- 最后错误
- 健康检查统计
- 状态变更记录

## 事件系统

支持的事件类型：
- Init: 服务初始化
- Start: 服务启动
- Stop: 服务停止
- Restart: 服务重启
- Error: 服务错误
- HealthCheck: 健康检查
- StateChange: 状态变更

## 最佳实践

查看 [examples/best_practice](examples/best_practice) 目录获取完整的最佳实践示例，包括：

1. 服务实现
   - 继承 BaseService 获取基本功能
   - 实现必要的生命周期钩子
   - 合理处理上下文取消
   - 根据服务重要性设置合适的优先级

2. 错误处理
   - 使用统一的错误类型
   - 正确设置错误码
   - 提供详细的错误信息

3. 依赖管理
   - 明确声明服务依赖
   - 避免循环依赖
   - 合理组织服务启动顺序

4. 监控和指标
   - 实现健康检查
   - 收集关键指标
   - 监控服务状态

## 文档

详细文档请访问 [pkg.go.dev](https://pkg.go.dev/github.com/darkit/service)

## 贡献

欢迎提交 Pull Request 和 Issue。

## 许可证

MIT License
