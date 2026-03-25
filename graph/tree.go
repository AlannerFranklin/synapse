package graph

import (
	"context"
	"fmt"
	"sync"

	"github.com/AlannerFranklin/synapse/schema"
)


// ==========================================
// Phase 3 核心：Blueprint Tree 蓝图树
// ==========================================

// TreeNode 表示树上的一个节点

type TreeNode struct {
	ID   string // 节点的唯一标识符
	Name string // 节点名称（比如 "LLM_Thinking"）

	// 双向指针：这是多叉树的核心！
	Parent   *TreeNode   // 指向父节点（支持回溯/回滚）
	Children []*TreeNode // 指向子节点列表（支持多分支推演）

	// 节点的具体执行逻辑
	RunFunc NodeFunc

	// 状态快照：该节点执行完毕后的状态备份
	// 如果用户想从这个节点重新分叉，就直接取这个快照
	Snapshot *schema.State
}

// Tree 表示整个蓝图执行树
type Tree struct {
	mu sync.RWMutex

	Root    *TreeNode            // 树的根节点
	NodeMap map[string]*TreeNode // 方便通过 ID 快速查找节点
}

// NewTree 创建一棵新的蓝图树
func NewTree(root *TreeNode) *Tree {
	t := &Tree {
		Root:    root,
		NodeMap: make(map[string]*TreeNode),
	}
	if root != nil {
		t.NodeMap[root.ID] = root
	}
	return t
}

// AddChild 为指定的父节点添加一个子节点（分叉）
func (t *Tree) AddChild(parentID string, child *TreeNode) error {
	t.mu.Lock()
	defer t.mu.Unlock()

	parent, exists := t.NodeMap[parentID]
	if !exists {
		return fmt.Errorf("parent node %s not found", parentID)
	}
	// 建立双向绑定
	child.Parent = parent
	parent.Children = append(parent.Children, child)
	t.NodeMap[child.ID] = child

	return nil
}

// ==========================================
// Phase 3 新增：非递归的树执行引擎
// ==========================================

// Run 从指定的节点开始向下执行整棵树。
// 我们使用非递归的“队列 (Queue)”方式，防止树太深导致爆栈。

func (t *Tree) Run(ctx context.Context, startNode *TreeNode, initialState *schema.State) error {
	if startNode == nil {
		return fmt.Errorf("start node cannot be nil")
	}
	// 1. 初始化执行队列 (Queue)
	// 队列里存的是即将要执行的节点，以及传给它的状态快照
	type task struct {
		node  *TreeNode
		state *schema.State
	}
	queue := []task{{node: startNode, state: initialState}}

	// 2. 开始非递归循环调度 (BFS 广度优先)
	for len(queue) > 0 {
		// 取出当前层级的所有任务
		levelSize := len(queue)
		var currentLevelTasks []task
		for i := 0; i < levelSize; i++ {
			currentLevelTasks = append(currentLevelTasks, queue[i])
		}
		queue = queue[levelSize:] // 切片前移

		// 我们使用 goroutine 并发执行这一层的所有节点！
		var wg sync.WaitGroup
		errCh := make(chan error, len(currentLevelTasks))

		for _, taskItem := range currentLevelTasks {
			wg.Add(1)
			go func(currentTask task) {
				defer wg.Done()

				if err := ctx.Err(); err != nil {
					errCh <- fmt.Errorf("tree execution canceled: %w", err)
					return
				}

				currNode := currentTask.node
				currState := currentTask.state

				fmt.Printf("▶ 正在执行蓝图节点: [%s] (ID: %s)\n", currNode.Name, currNode.ID)
				currState.AddTrace(currNode.Name, "NodeStart", "开始执行蓝图节点", nil)

				// 3. 执行当前节点的业务逻辑
				if currNode.RunFunc != nil {
					if err := currNode.RunFunc(ctx, currState); err != nil {
						currState.AddTrace(currNode.Name, "NodeError", fmt.Sprintf("执行失败: %v", err), nil)
						errCh <- fmt.Errorf("node [%s] failed: %w", currNode.Name, err)
						return
					}
				}
				currState.AddTrace(currNode.Name, "NodeSuccess", "节点执行成功", nil)

				// 4. 保存当前节点的状态快照！
				currNode.Snapshot = currState.Clone()

				// 5. 把当前状态里的新数据写回到传入的根状态中 (全局黑板)
				for k, v := range currState.Data {
					initialState.SetData(k, v)
				}
				
				// 修复专家节点记忆滞后问题：将当前节点的私有消息也合并到全局状态中
				initialState.SetMessages(currState.GetMessages())
			}(taskItem)
		}

		// 等待这一层所有节点执行完毕
		wg.Wait()
		close(errCh)

		// 检查是否有节点报错
		for err := range errCh {
			if err != nil {
				return err
			}
		}

		// 6. 这一层执行完毕且状态合并后，把所有子节点加入下一层的队列
		// 注意：为了避免同一个子节点被多次加入（比如多个专家指向同一个评估节点），我们需要去重
		nextLevelNodes := make(map[string]*TreeNode)
		for _, currentTask := range currentLevelTasks {
			for _, child := range currentTask.node.Children {
				nextLevelNodes[child.ID] = child
			}
		}

		for _, child := range nextLevelNodes {
			// 传递给子节点的是合并后的最新的全局 state 的拷贝！
			childState := initialState.Clone() 
			queue = append(queue, task{
				node:  child,
				state: childState,
			})
		}
	}

	return nil
}