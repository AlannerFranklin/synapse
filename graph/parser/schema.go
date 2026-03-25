package parser

// ==========================================
// 蓝图配置文件 (DSL) 解析器数据结构
// ==========================================

// BlueprintConfig 代表整个蓝图配置文件的顶层结构
type BlueprintConfig struct {
	Name        string       `json:"name"`        // 蓝图名称
	Description string       `json:"description"` // 蓝图描述
	Nodes       []NodeConfig `json:"nodes"`       // 所有的节点定义
}

// NodeConfig 代表配置文件中单个节点的定义
type NodeConfig struct {
	ID       string   `json:"id"`       // 节点唯一标识
	Name     string   `json:"name"`     // 节点名称
	Type     string   `json:"type"`     // 节点类型，如 "llm_think", "evaluator", "memory_load"
	
	// --- 新增：动态参数配置区 ---
	// 语义单位 (Prompt 模板)
	SystemPrompt string `json:"system_prompt,omitempty"` 
	
	// 状态边界 (输入/输出的 Key 绑定)
	InputKeys  []string `json:"input_keys,omitempty"`  // 该节点需要从 State 中读取哪些变量
	OutputKey  string   `json:"output_key,omitempty"`  // 该节点的结果要写入 State 的哪个 Key

	Children []string `json:"children"` // 子节点 ID 列表，用于构建树结构
}
