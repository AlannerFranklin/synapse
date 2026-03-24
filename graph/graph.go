package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/AlannerFranklin/synapse/schema"
)

// Graph 定义了整个执行图
type Graph struct {
	// Nodes 存放按顺序执行的节点列表
	Nodes []*Node
}

func NewGraph() *Graph {
	return &Graph{
		Nodes: make([]*Node, 0),
	}
}

// AddNode 往图的末尾追加一个节点
func (g *Graph) AddNode(node *Node) {
	g.Nodes = append(g.Nodes, node)
}

// ==========================================
// 核心逻辑：运行整个图
// ==========================================

func (g *Graph) Run(ctx context.Context, state *schema.State) error {
	// 遍历图里的每一个节点，按顺序执行
	for _, node := range g.Nodes {
		// 判断：用户有没有通过 ctx 中途取消任务？
		// 比如网页被关掉了，我们就没必要继续执行后续节点了。
		if err := ctx.Err(); err != nil {
			return fmt.Errorf("graph execution canceled: %w", err)
		}
		fmt.Printf("▶ 正在执行节点: [%s]\n", node.Name)
		// 根据节点类型，决定怎么执行
		if node.Type == NodeTypeNormal {
			// ============================
			// 串行执行
			// ============================
			if err := node.Run(ctx, state); err != nil {
				return fmt.Errorf("node [%s] failed: %w", node.Name, err)
			}
		} else if node.Type == NodeTypeParallel {
			// ============================
			// 并行执行 (Verilog思维启动！)
			// ============================
			if err := g.runParallel(ctx, state, node); err != nil {
				return fmt.Errorf("parallel node [%s] failed: %w", node.Name, err)
			}
		}
	}
	return nil
}

func (g *Graph) runParallel(ctx context.Context, state *schema.State, node *Node) error {
	// 1. WaitGroup (等待组)
	// 就像是带队老师，出去 3 个学生，老师必须等这 3 个学生都回来，才能继续走。
	var wg sync.WaitGroup
	// 2. Channel (管道)
	// 如果某个并行任务出错了，我们需要把错误传回主线程。
	// channel 就像是一根管子，容量我们设为并行任务的数量。
	errCh := make(chan error, len(node.ParallelRuns))
	// 遍历所有的并行任务
	for i, runFunc := range node.ParallelRuns {
		// 每派出去一个学生，老师计数器 +1
		wg.Add(1)

		// 3. Goroutine (开启并发！)
		// `go` 关键字就像硬件里的并发触发，它会瞬间开启一个全新的线程（协程）去执行大括号里的代码。
		// 主线程不会等它，而是瞬间进入下一次 for 循环。
		go func(index int, fn NodeFunc) {
			// defer wg.Done()：不管这个学生是正常完成还是摔跤了，结束时必须告诉老师 "我回来了" (计数器 -1)
			defer wg.Done()
			// 执行真正的业务逻辑
			if err := fn(ctx, state); err != nil {
				// 如果报错了，把错误塞进管子里
				errCh <- fmt.Errorf("task %d failed: %w", index, err)
			}
		}(i, runFunc) // 把 i 和 runFunc 传进闭包里（防止变量捕获坑）
	}

	// 老师站在这里死等，直到所有学生都执行完 defer wg.Done()
	wg.Wait()
	// 当代码运行到这里，说明所有并发任务都已经结束了！关闭管子。
	close(errCh)

	// 检查管子里有没有错误
	// 我们遍历管子，只要发现有任何一个任务报错了，我们就宣告整个并行节点失败。
	for err := range errCh {
		if err != nil {
			return err
		}
	}
	return nil
}