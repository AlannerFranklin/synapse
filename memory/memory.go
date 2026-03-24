package memory

import (
	"sync"

	"github.com/AlannerFranklin/synapse/schema"
)

// ==========================================
// 1. 定义记忆接口
// ==========================================
// 无论是什么记忆（基于内存、基于 Redis、基于向量库），
// 都必须实现这三个方法。

type Memory interface {
	// AddMessage 添加一条新消息到记忆中
	AddMessage(msg schema.Message)
	// GetMessages 获取当前所有应该发给大模型的消息
	GetMessages() []schema.Message
	// Clear 清空记忆
	Clear()
}

// ==========================================
// 2. 实现短期记忆 (滑动窗口)
// ==========================================

// ShortTermMemory 是一种基于内存的滑动窗口记忆。
// 它只保留最近的 N 条消息，防止 Token 溢出。

type ShortTermMemory struct {
	mu       sync.RWMutex
	messages []schema.Message
	maxSize  int // 最大保留的消息数量
}

// NewShortTermMemory 创建一个短期记忆实例
// maxSize: 比如设为 10，代表只记住最近的 10 句话
func NewShortTermMemory(maxSize int) *ShortTermMemory {
	return &ShortTermMemory{
		messages: make([]schema.Message, 0),
		maxSize:  maxSize,
	}
}

// AddMessage 添加新消息，并维护滑动窗口
func (m *ShortTermMemory) AddMessage(msg schema.Message) {
	m.mu.Lock()
	defer m.mu.Unlock()

	// 把新消息加到最后
	m.messages = append(m.messages, msg)

	// ==========================================
	// 语法教学：切片截取 (Slicing)
	// ==========================================
	// 如果消息总数超过了 maxSize，我们需要把最前面的旧消息"切"掉。
	// 在 Go 里面，切片的语法是 array[startIndex : endIndex]
	// - startIndex 包含
	// - endIndex 不包含
	// 比如 array[2:] 意思是从索引 2 一直取到末尾，前两个元素就丢掉了。
	if len(m.messages) > m.maxSize {
		// 计算超出了多少条
		overflow := len(m.messages) - m.maxSize
		// 截取掉旧的，保留新的
		m.messages = m.messages[overflow:]
	}
}

func (m *ShortTermMemory) GetMessages() []schema.Message {
	m.mu.RLock()
	defer m.mu.RUnlock()

	// 浅拷贝一份返回，防止外部直接修改
	res := make([]schema.Message, len(m.messages))
	copy(res, m.messages)
	return res
}

// Clear 清空所有记忆
func (m *ShortTermMemory) Clear() {
	m.mu.Lock()
	defer m.mu.Unlock()
	// 把切片重新变成一个空切片
	m.messages = make([]schema.Message, 0)
}