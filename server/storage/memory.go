package storage

import (
	"basic_c2/internal/models"
	"sync"
)

// Memory 内存数据库，存储在线主机和命令队列
type Memory struct {
	CommandQueue map[string]string           // 待发命令队列 [AgentID]Command
	Agents       map[string]*models.AgentInfo // 在线主机列表 [AgentID]Info
	mutex        *sync.Mutex                 // 线程安全锁
}

// NewMemory 创建新的内存数据库实例
func NewMemory() *Memory {
	return &Memory{
		CommandQueue: make(map[string]string),
		Agents:       make(map[string]*models.AgentInfo),
		mutex:        &sync.Mutex{},
	}
}

// Lock 加锁
func (m *Memory) Lock() {
	m.mutex.Lock()
}

// Unlock 解锁
func (m *Memory) Unlock() {
	m.mutex.Unlock()
}

// GetAgent 获取指定主机信息
func (m *Memory) GetAgent(id string) (*models.AgentInfo, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	agent, exists := m.Agents[id]
	return agent, exists
}

// UpdateAgent 更新或添加主机信息
func (m *Memory) UpdateAgent(agent *models.AgentInfo) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.Agents[agent.ID] = agent
}

// GetAllAgents 获取所有主机列表
func (m *Memory) GetAllAgents() []*models.AgentInfo {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	list := make([]*models.AgentInfo, 0, len(m.Agents))
	for _, agent := range m.Agents {
		list = append(list, agent)
	}
	return list
}

// EnqueueCommand 添加待发送命令
func (m *Memory) EnqueueCommand(agentID, command string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	m.CommandQueue[agentID] = command
}

// DequeueCommand 取出待发送命令（取出后删除）
func (m *Memory) DequeueCommand(agentID string) (string, bool) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	cmd, exists := m.CommandQueue[agentID]
	if exists {
		delete(m.CommandQueue, agentID)
	}
	return cmd, exists
}

// DeleteAgent 删除主机记录
func (m *Memory) DeleteAgent(agentID string) {
	m.mutex.Lock()
	defer m.mutex.Unlock()
	
	delete(m.Agents, agentID)
	delete(m.CommandQueue, agentID)
}
