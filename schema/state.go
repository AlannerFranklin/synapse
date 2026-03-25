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

type TraceLog struct {
	Timestamp time.Time // 记录发生的时间
	NodeName  string    // 是哪个节点触发的（比如 "SearchNode"）
	Action    string    // 做了什么动作（比如 "调用搜索引擎"）
	Reasoning string    // 为什么这么做（大模型的思考过程）
	Result    any       // 执行的结果（可以是字符串，也可以是报错信息）
}

// State 表示在 DAG（有向无环图）执行过程中，在各个节点之间流转的全局状态。
// 它必须是并发安全的，因为可能会有多个节点并行运行并同时修改它。

type State struct {
	mu sync.RWMutex

	// Messages 记录了这一轮图执行过程中的所有对话消息
	Messages []Message

	// Data 用于存放各个节点产生的一些中间数据（比如搜索结果、计算结果）
	Data map[string]any

	// Traces 存放所有的执行轨迹（黑匣子记录）
	Traces []TraceLog
}


func NewState() *State {
	return &State{
		Messages: make([]Message, 0),
		Data:     make(map[string]any),
		Traces:   make([]TraceLog, 0), // 初始化轨迹切片
	}
}

// SetData 往状态里写入中间数据（并发安全）
func (s *State) SetData(key string, value any) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Data[key] = value
}

// GetData 从状态里读取中间数据（并发安全）
func (s *State) GetData(key string) (any, bool) {
	s.mu.RLock()
	defer s.mu.RUnlock()
	val, ok := s.Data[key]
	return val, ok
}

// AddMessage 向当前状态的短期记忆中追加消息
func (s *State) AddMessage(msg Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	
	// 简单的滑动窗口逻辑：最多保留 10 条
	s.Messages = append(s.Messages, msg)
	if len(s.Messages) > 10 {
		s.Messages = s.Messages[len(s.Messages)-10:]
	}
}

// GetMessages 获取当前所有的消息（并发安全）
func (s *State) GetMessages() []Message {
	s.mu.RLock()
	defer s.mu.RUnlock()

	msgs := make([]Message, len(s.Messages))
	copy(msgs, s.Messages)
	return msgs
}

// SetMessages 直接覆盖当前的消息列表（用于状态同步）
func (s *State) SetMessages(msgs []Message) {
	s.mu.Lock()
	defer s.mu.Unlock()
	s.Messages = make([]Message, len(msgs))
	copy(s.Messages, msgs)
}

// ==========================================
// Phase 2 新增：轨迹记录方法
// ==========================================

// AddTrace 添加一条执行轨迹（并发安全）

func (s *State) AddTrace(nodeName, action, reasoning string, result any) {
	s.mu.Lock()
	defer s.mu.Unlock()

	s.Traces = append(s.Traces, TraceLog{
		Timestamp: time.Now(), // 自动打上当前时间戳
		NodeName:  nodeName,
		Action:    action,
		Reasoning: reasoning,
		Result:    result,
	})
}
// PrintTraces 打印整个图的执行回放记录（非常酷炫的终端输出）
func (s *State) PrintTraces() {
	s.mu.RLock()
	defer s.mu.RUnlock()

	fmt.Println("\n==========================================")
	fmt.Println("⏪ [Agent 决策回放 (Trace Replay)]")
	fmt.Println("==========================================")

	if len(s.Traces) == 0 {
		fmt.Println("没有记录到任何执行轨迹。")
		return
	}
	for i, trace := range s.Traces {
		// 格式化时间，比如 "15:04:05.000"
		timeStr := trace.Timestamp.Format("15:04:05.000")
		fmt.Printf("[%d] ⏰ 时间: %s | 📍 节点: %s\n", i+1, timeStr, trace.NodeName)
		
		if trace.Reasoning != "" {
			fmt.Printf("   💡 思考: %s\n", trace.Reasoning)
		}
		
		fmt.Printf("   🎯 动作: %s\n", trace.Action)
		
		if trace.Result != nil {
			fmt.Printf("   📄 结果: %v\n", trace.Result)
		}
		fmt.Println("------------------------------------------")
	}
}

// ==========================================
// Phase 3 新增：蓝图引擎核心 - 状态快照 (Deep Copy)
// ==========================================

// Clone 深度拷贝当前的状态，返回一个全新的 State 实例。
// 这是实现“时空穿梭”和“多叉树分支”的核心！
func (s *State) Clone() *State {
	s.mu.RLock()
	defer s.mu.RUnlock()

	// 1. 创建一个新的空 State
	newState := NewState()

	// 2. 拷贝 Messages (切片的深拷贝)
	newState.Messages = make([]Message, len(s.Messages))
	copy(newState.Messages, s.Messages)

	// 3. 拷贝 Data (Map 的深拷贝)
	// 注意：这里我们假设 value 是基本类型（string, int 等）。
	// 如果 value 里存了复杂的指针，这里还需要更深层的反射拷贝，但目前这样足够了。
	for k, v := range s.Data {
		newState.Data[k] = v
	}
	// 4. 拷贝 Traces (黑匣子记录的深拷贝)
	newState.Traces = make([]TraceLog, len(s.Traces))
	copy(newState.Traces, s.Traces)
	return newState
}