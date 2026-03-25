package parser

import (
	"context"
	"encoding/json"
	"fmt"
	"os"
	"strings"

	"github.com/AlannerFranklin/synapse/graph"
	"github.com/AlannerFranklin/synapse/llm"
	"github.com/AlannerFranklin/synapse/schema"
)

// ==========================================
// 蓝图解析器 (Parser) 核心逻辑
// ==========================================

// ParseFromFile 从指定的 JSON 文件加载并构建 Blueprint Tree
func ParseFromFile(filePath string, provider llm.Model, dbPath string) (*graph.Tree, error) {
	// 1. 读取 JSON 文件内容
	data, err := os.ReadFile(filePath)
	if err != nil {
		return nil, fmt.Errorf("failed to read blueprint file: %w", err)
	}

	// 2. 将 JSON 解析为 BlueprintConfig 结构体
	var config BlueprintConfig
	if err := json.Unmarshal(data, &config); err != nil {
		return nil, fmt.Errorf("failed to parse blueprint json: %w", err)
	}

	// 3. 第一次遍历：创建所有节点并放入 Map 中
	// 因为后面要组装树，所以先用一个字典把所有生成的 TreeNode 存起来
	nodeMap := make(map[string]*graph.TreeNode)

	for _, nc := range config.Nodes {
		node := createNodeFromConfig(nc, provider, dbPath)
		nodeMap[nc.ID] = node
	}

	// 4. 第二次遍历：根据 children 字段组装树形结构
	var rootNode *graph.TreeNode
	
	// 这里我们做一个简单的假设：数组里的第一个节点就是 Root 节点
	if len(config.Nodes) > 0 {
		rootNode = nodeMap[config.Nodes[0].ID]
	}

	tree := graph.NewTree(rootNode)
	
	// 把其他节点也注册到树的 NodeMap 里，方便 AddChild 查找
	for id, node := range nodeMap {
		if id != rootNode.ID {
			tree.NodeMap[id] = node
		}
	}

	// 建立父子关系
	for _, nc := range config.Nodes {
		for _, childID := range nc.Children {
			childNode, exists := nodeMap[childID]
			if !exists {
				return nil, fmt.Errorf("child node %s not found for parent %s", childID, nc.ID)
			}
			err := tree.AddChild(nc.ID, childNode)
			if err != nil {
				return nil, fmt.Errorf("failed to add child %s to %s: %w", childID, nc.ID, err)
			}
		}
	}

	fmt.Printf("✅ 成功加载蓝图: %s\n", config.Name)
	return tree, nil
}

// createNodeFromConfig 根据配置中的 Type 字段，动态生成对应的执行逻辑
func createNodeFromConfig(nc NodeConfig, provider llm.Model, dbPath string) *graph.TreeNode {
	node := &graph.TreeNode{
		ID:   nc.ID,
		Name: nc.Name,
	}

	// ==========================================
	// 核心：工厂模式！根据字符串生成具体的函数逻辑
	// ==========================================
	switch nc.Type {
	case "memory_load":
		node.RunFunc = func(ctx context.Context, state *schema.State) error {
			content, err := os.ReadFile(dbPath)
			var longTermStr string
			if err == nil && len(content) > 0 {
				longTermStr = string(content)
			} else {
				longTermStr = "暂无。"
			}
			state.SetData("long_term_knowledge", longTermStr)
			return nil
		}

	case "llm_think":
		// 现在这是一个完全通用的 LLM 节点，它不绑定任何特定的业务逻辑
		node.RunFunc = func(ctx context.Context, state *schema.State) error {
			// 1. 根据配置文件中的 InputKeys，从状态机中提取依赖数据
			var inputs []interface{}
			for _, key := range nc.InputKeys {
				val, _ := state.GetData(key)
				inputs = append(inputs, val)
			}

			// 2. 动态组装 SystemPrompt (利用 Go 的 fmt.Sprintf 和动态参数)
			sysPrompt := nc.SystemPrompt
			if len(inputs) > 0 {
				// 如果 Prompt 里有 %v 占位符，我们就把输入填进去
				sysPrompt = fmt.Sprintf(nc.SystemPrompt, inputs...)
			}

			// 强制输出 JSON 格式
			sysPrompt += `
【强制要求】
你必须且只能返回一个 JSON 格式的字符串，不要输出 json 标记。
JSON 的结构必须如下：
{
  "thought": "你的思考过程",
  "response": "你最终要回答的话"
}`

			// 从当前状态机的独立记忆中获取历史消息
			finalMessages := []schema.Message{{Role: schema.RoleSystem, Content: sysPrompt}}
			finalMessages = append(finalMessages, state.Messages...)
			
			// 如果输入里有用户最新的问题
			userInput, _ := state.GetData("user_input")
			finalMessages = append(finalMessages, schema.Message{Role: schema.RoleUser, Content: userInput.(string)})

			// 3. 调用大模型
			resp, err := provider.Generate(ctx, finalMessages, &llm.GenerateOptions{Temperature: 0.8})
			if err != nil {
				return err
			}

			// 4. 解析结果
			type LLMResponse struct {
				Thought  string `json:"thought"`
				Response string `json:"response"`
			}
			
			// 清理大模型可能返回的 markdown 标记
			cleanJSON := strings.TrimSpace(resp.Content)
			if strings.HasPrefix(cleanJSON, "```json") {
				cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
			} else if strings.HasPrefix(cleanJSON, "```") {
				cleanJSON = strings.TrimPrefix(cleanJSON, "```")
			}
			if strings.HasSuffix(cleanJSON, "```") {
				cleanJSON = strings.TrimSuffix(cleanJSON, "```")
			}
			cleanJSON = strings.TrimSpace(cleanJSON)

			var parsedResp LLMResponse
			if err := json.Unmarshal([]byte(cleanJSON), &parsedResp); err != nil {
				// 如果解析失败，回退为直接把原始内容存入
				parsedResp.Response = resp.Content
				parsedResp.Thought = "解析JSON失败"
			}

			// 5. 将结果写入配置文件指定的 OutputKey 中
			if nc.OutputKey != "" {
				state.SetData(nc.OutputKey, parsedResp.Response)
			}
			
			// 记录执行轨迹
			state.AddTrace(nc.Name, "通用思考节点", parsedResp.Thought, parsedResp.Response)
			return nil
		}

	case "evaluator":
		// 评估节点也变得更通用，它读取多个 InputKeys 作为候选答案
		node.RunFunc = func(ctx context.Context, state *schema.State) error {
			userInput, _ := state.GetData("user_input")
			
			// 动态收集依赖的分支答案
			var candidatesInfo string
			for i, key := range nc.InputKeys {
				val, _ := state.GetData(key)
				candidatesInfo += fmt.Sprintf("专家%d: %v\n", i+1, val)
			}

			sysPrompt := nc.SystemPrompt + `
【强制要求】
必须返回 JSON，格式如下：
{
  "thought": "你的评估过程",
  "best_expert": "你选择的专家",
  "final_response": "你最终决定输出给用户的回答"
}`
			userPrompt := fmt.Sprintf(`用户的问题是: "%s"

候选答案如下：
%s

请选出最好的回答！`, userInput, candidatesInfo)

			finalMessages := []schema.Message{
				{Role: schema.RoleSystem, Content: sysPrompt},
				{Role: schema.RoleUser, Content: userPrompt},
			}

			resp, err := provider.Generate(ctx, finalMessages, &llm.GenerateOptions{Temperature: 0.3})
			if err != nil {
				return err
			}

			type EvalResponse struct {
				Thought       string `json:"thought"`
				BestExpert    string `json:"best_expert"`
				FinalResponse string `json:"final_response"`
			}
			
			// 清理大模型可能返回的 markdown 标记
			cleanJSON := strings.TrimSpace(resp.Content)
			if strings.HasPrefix(cleanJSON, "```json") {
				cleanJSON = strings.TrimPrefix(cleanJSON, "```json")
			} else if strings.HasPrefix(cleanJSON, "```") {
				cleanJSON = strings.TrimPrefix(cleanJSON, "```")
			}
			if strings.HasSuffix(cleanJSON, "```") {
				cleanJSON = strings.TrimSuffix(cleanJSON, "```")
			}
			cleanJSON = strings.TrimSpace(cleanJSON)

			var parsedResp EvalResponse
			if err := json.Unmarshal([]byte(cleanJSON), &parsedResp); err != nil {
				parsedResp.Thought = "解析JSON失败: " + err.Error()
				parsedResp.BestExpert = "解析失败"
				parsedResp.FinalResponse = resp.Content // 回退为原始内容
			}

			fmt.Printf("\n🏆 [%s 评估]: %s\n", nc.Name, parsedResp.Thought)
			fmt.Printf("👑 [胜出者]: %s\n", parsedResp.BestExpert)
			fmt.Printf("🤖 AI: %s\n", parsedResp.FinalResponse)

			state.AddTrace(nc.Name, "评估与决断", parsedResp.Thought, parsedResp.FinalResponse)
			
			if nc.OutputKey != "" {
				state.SetData(nc.OutputKey, parsedResp.FinalResponse)
			}
			return nil
		}

	case "memory_save":
		node.RunFunc = func(ctx context.Context, state *schema.State) error {
			userInput, _ := state.GetData("user_input")
			aiResponse, ok := state.GetData("ai_response")
			
			if ok {
				// 将本轮对话存入状态机的独立记忆中
				state.AddMessage(schema.Message{Role: schema.RoleUser, Content: userInput.(string)})
				state.AddMessage(schema.Message{Role: schema.RoleAssistant, Content: aiResponse.(string)})
			}
			return nil
		}
	}

	return node
}