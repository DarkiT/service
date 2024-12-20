package service

import (
	"fmt"
	"sort"
	"sync"
)

// ServiceNode 表示依赖图中的服务节点
type ServiceNode struct {
	Name     string
	Priority ServicePriority
	Deps     []string
}

// DependencyGraph 管理服务依赖关系
type DependencyGraph struct {
	mu    sync.RWMutex
	nodes map[string]*ServiceNode
}

// NewDependencyGraph 创建新的依赖图
func NewDependencyGraph() *DependencyGraph {
	return &DependencyGraph{
		nodes: make(map[string]*ServiceNode),
	}
}

// AddNode 添加服务节点
func (dg *DependencyGraph) AddNode(node *ServiceNode) error {
	dg.mu.Lock()
	defer dg.mu.Unlock()

	// 检查是否已存在
	if _, exists := dg.nodes[node.Name]; exists {
		return &ServiceError{
			Code:    ErrServiceAlreadyExists,
			Message: fmt.Sprintf("service %s already exists in dependency graph", node.Name),
		}
	}

	// 检查是否会形成循环依赖
	if err := dg.checkCyclicDependency(node.Name, node.Deps); err != nil {
		return err
	}

	dg.nodes[node.Name] = node
	return nil
}

// checkCyclicDependency 检查是否存在循环依赖
func (dg *DependencyGraph) checkCyclicDependency(service string, deps []string) error {
	visited := make(map[string]bool)
	var check func(string) error

	check = func(current string) error {
		if current == service {
			return &ServiceError{
				Code:    ErrDependencyFailed,
				Message: "cyclic dependency detected",
			}
		}
		if visited[current] {
			return nil
		}
		visited[current] = true

		node, exists := dg.nodes[current]
		if exists {
			for _, dep := range node.Deps {
				if err := check(dep); err != nil {
					return err
				}
			}
		}
		return nil
	}

	for _, dep := range deps {
		if err := check(dep); err != nil {
			return err
		}
	}
	return nil
}

// GetStartOrder 获取服务启动顺序
func (dg *DependencyGraph) GetStartOrder() ([]string, error) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	visited := make(map[string]bool)
	temp := make(map[string]bool)
	var order []*ServiceNode

	var visit func(string) error
	visit = func(name string) error {
		if temp[name] {
			return &ServiceError{
				Code:    ErrDependencyFailed,
				Message: fmt.Sprintf("cyclic dependency detected involving %s", name),
			}
		}
		if visited[name] {
			return nil
		}
		temp[name] = true

		node := dg.nodes[name]
		for _, dep := range node.Deps {
			if err := visit(dep); err != nil {
				return err
			}
		}
		temp[name] = false
		visited[name] = true
		order = append(order, node)
		return nil
	}

	// 遍历所有节点
	for name := range dg.nodes {
		if !visited[name] {
			if err := visit(name); err != nil {
				return nil, err
			}
		}
	}

	// 按层级和优先级排序
	levels := make(map[int][]*ServiceNode)
	maxLevel := 0

	// 计算每个服务的层级
	for i, node := range order {
		level := 0
		for _, dep := range node.Deps {
			for j := 0; j < i; j++ {
				if order[j].Name == dep && j > level {
					level = j
				}
			}
		}
		if level > maxLevel {
			maxLevel = level
		}
		levels[level] = append(levels[level], node)
	}

	// 对每个层级内的服务按优先级排序
	result := make([]string, 0, len(order))
	for level := 0; level <= maxLevel; level++ {
		services := levels[level]
		if len(services) > 0 {
			// 同层级服务按优先级排序
			sort.Slice(services, func(i, j int) bool {
				return services[i].Priority < services[j].Priority
			})
			for _, s := range services {
				result = append(result, s.Name)
			}
		}
	}

	return result, nil
}

// GetDependencies 获取服务的依赖
func (dg *DependencyGraph) GetDependencies(name string) ([]string, bool) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	node, exists := dg.nodes[name]
	if !exists {
		return nil, false
	}
	return node.Deps, true
}

// GetNode 获取服务节点
func (dg *DependencyGraph) GetNode(name string) (*ServiceNode, bool) {
	dg.mu.RLock()
	defer dg.mu.RUnlock()

	node, exists := dg.nodes[name]
	if !exists {
		return nil, false
	}
	return node, true
}
