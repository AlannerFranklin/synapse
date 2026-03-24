/*
package schema

import "sync"

// ==========================================
// 语法教学：Go 的 Map 和 any 类型
// ==========================================
// map[string]any 是 Go 里的字典类型：
// - string 是键 (Key) 的类型
// - any 是值 (Value) 的类型。any 是 interface{} 的别名，代表"可以是任何类型"。
// 相当于 Python 里的 dict，或者 Java 里的 Map<String, Object>。

// State 表示在 DAG（有向无环图）执行过程中，在各个节点之间流转的全局状态。
// 它必须是并发安全的，因为可能会有多个节点并行运行并同时修改它。

type State struct {
	// ==========================================
	// 语法教学：并发控制 (sync.RWMutex)
	// ==========================================
	// RWMutex 是读写锁。
	// - 当多个协程（goroutine）只想读取数据时，它们可以同时读。
	// - 当某个协程想要修改数据时，它必须"加锁"，此时其他协程既不能读也不能写，直到它"解锁"。
	// 这保证了并发情况下的数据安全。
	mu sync.RWMutex

	// Messages 记录了这一轮图执行过程中的所有对话消息
	Messages []Message

	// Data 用于存放各个节点产生的一些中间数据（比如搜索结果、计算结果）
	Data map[string]any
}

// ==========================================
// 语法教学：初始化函数 (Constructor 模式)
// ==========================================
// Go 没有 class，所以也没有内置的构造函数（constructor）。
// 惯例是写一个叫 NewXXX 的普通函数来初始化并返回结构体的指针。
// 为什么 Map 需要被 make 初始化？
// - 在 Go 中，只声明 map 不分配内存，它的值是 nil。直接往 nil map 里写数据会触发 panic（崩溃）。
// - 必须用 make(map[string]any) 来分配内存。

func NewState() *State {
	return &State{
		Messages: make([]Message, 0),
		Data:     make(map[string]any),
	}
}

// SetData 往状态里写入中间数据（并发安全）
func (s *State) SetData(key string, value any) {
	s.mu.Lock()         // 加写锁：独占访问
	defer s.mu.Unlock() // defer：不管函数是怎么退出的（正常结束还是报错），在函数结束前一定会执行这行代码（解锁）。非常实用！

	s.Data[key] = value
}

// GetData 从状态里读取中间数据（并发安全）
func (s *State) GetData(key string) (any, bool) {
	s.mu.RLock()         // 加读锁：允许多个协程同时读
	defer s.mu.RUnlock() // 退出时解除读锁

	// ==========================================
	// 语法教学：Map 的安全读取 (Comma Ok 惯用法)
	// ==========================================
	// 从 map 取值时，可以返回两个变量。
	// 第一个是取到的值 (val)，第二个是一个布尔值 (ok)。
	// 如果 key 存在，ok 就是 true；如果 key 不存在，ok 就是 false，并且 val 是默认空值。
	val, ok := s.Data[key]
	return val, ok
}

// GetMessages 获取当前所有的消息（并发安全）
func (s *State) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 浅拷贝一份切片返回，防止外部直接修改底层数组
	msgs := make([]Message, len(s.Messages))
	copy(msgs, s.Messages)
	return msgs
}
*/

package schema

import (
	"fmt"
	"sync"
	"time"
)

// ==========================================
// Phase 2 新增：决策回放记录 (TraceLog)
// ==========================================

// TraceLog 记录了 Agent 在图执行过程中的一个关键动作。
// 就像飞机的黑匣子，它能让我们事后回放 Agent 的思考和执行过程。