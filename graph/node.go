package graph

import (
	"context"

	"github.com/AlannerFranklin/synapse/schema"
)

// ==========================================
// 语法教学：定义函数的别名类型
// ==========================================
// NodeFunc 是图里面每个节点实际执行的业务逻辑。
// 它接收当前的全局状态 (*schema.State)，你可以读取状态，也可以修改状态。
// 它返回一个 error，如果返回的 error 不为空，整个图的执行就会终止。
type NodeFunc func(ctx context.Context, state *schema.State) error

// ==========================================
// 语法教学：枚举 (Enum) 模拟
// ==========================================
// Go 语言没有 enum 关键字，通常用自定义类型 + 常量来模拟枚举。

type NodeType string

const (
	NodeTypeNormal   NodeType = "normal"   // 普通的串行节点
	NodeTypeParallel NodeType = "parallel" // 并行节点（这个节点内部可以包含多个并发执行的子函数）
)

// Node 代表图中的一个执行节点
type Node struct {
	Name string   // 节点的名字，比如 "search_node"
	Type NodeType // 节点的类型

	// 如果是普通节点，执行这个函数
	Run NodeFunc

	// 如果是并行节点，我们会同时启动多个 Goroutine 去执行这组函数
	ParallelRuns []NodeFunc
}

// ==========================================
// 语法教学：可变参数 (Variadic Parameters)
// ==========================================
// 注意参数里的 `funcs ...NodeFunc`。
// 这代表你可以传 1 个、2 个甚至 100 个 NodeFunc 进来，
// 在函数内部，funcs 会变成一个切片 []NodeFunc。
// 
// NewParallelNode 创建一个并行节点
func NewParallelNode(name string, funcs ...NodeFunc) *Node {
	return &Node{
		Name:         name,
		Type:         NodeTypeParallel,
		ParallelRuns: funcs,
	}
}

// NewNode 创建一个普通的串行节点
func NewNode(name string, run NodeFunc) *Node {
	return &Node{
		Name: name,
		Type: NodeTypeNormal,
		Run:  run,
	}
}