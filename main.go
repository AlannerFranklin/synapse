package main

import (
	"bufio"
	"context"
	"encoding/json" // 👈 新增：用于解析 JSON
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AlannerFranklin/synapse/graph"
	"github.com/AlannerFranklin/synapse/llm"
	"github.com/AlannerFranklin/synapse/memory"
	"github.com/AlannerFranklin/synapse/schema"
)

func main() {
	fmt.Println("========================================")
	fmt.Println("🧠 Synapse Agent 终极形态已启动！")
	fmt.Println("========================================")

	// 1. 初始化核心组件
	// 读取本地 API Key 文件
	apiKeyBytes, err := os.ReadFile("api.txt")
	if err != nil {
		log.Fatalf("❌ 无法读取 api.txt 文件: %v\n请在根目录创建 api.txt 并填入你的 API Key", err)
	}
	apiKey := strings.TrimSpace(string(apiKeyBytes))
	
	provider := llm.NewOpenAIProvider("https://api.deepseek.com/v1", apiKey, "deepseek-chat")
	shortMem := memory.NewShortTermMemory(6) // 短期记忆：记住最近 6 句话
	dbPath := "long_term_memory.txt"         // 长期记忆文件路径

	// 2. 构建 Agent 蓝图树 (Blueprint Tree)
	// 我们先定义三个节点，然后再把它们像拼图一样拼起来！

	// 节点 A: 提取长期记忆 (作为树的根节点 Root)
	nodeLoadMemory := &graph.TreeNode {
		ID:   "node_root_01",
		Name: "LoadLongTermMemory",
		RunFunc: func(ctx context.Context, state *schema.State) error {
			content, err := os.ReadFile(dbPath)
			var longTermStr string
			if err == nil && len(content) > 0 {
				longTermStr = string(content)
			} else {
				longTermStr = "暂无。"
			}
			state.SetData("long_term_knowledge", longTermStr)
			return nil
		},
	}
	// 节点 B: 大模型核心思考
	nodeLLMThinking := &graph.TreeNode{
		ID:   "node_llm_02",
		Name: "LLM_Thinking",
		RunFunc: func(ctx context.Context, state *schema.State) error {
			userInput, _ := state.GetData("user_input")
			longTermKnowledge, _ := state.GetData("long_term_knowledge")

			sysPrompt := fmt.Sprintf(`你是一个聪明的 AI 助手。
这是你对用户的长期记忆：[%s]

【强制要求】
你必须且只能返回一个 JSON 格式的字符串，不要输出 json 标记。
JSON 的结构必须如下：
{
  "thought": "你的思考过程",
  "response": "你最终要回答的话"
}`, longTermKnowledge)

			finalMessages := []schema.Message{{Role: schema.RoleSystem, Content: sysPrompt}}
			finalMessages = append(finalMessages, shortMem.GetMessages()...)
			finalMessages = append(finalMessages, schema.Message{Role: schema.RoleUser, Content: userInput.(string)})

			resp, err := provider.Generate(ctx, finalMessages, &llm.GenerateOptions{Temperature: 0.7})
			if err != nil {
				return err
			}

			type LLMResponse struct {
				Thought  string `json:"thought"`
				Response string `json:"response"`
			}
			var parsedResp LLMResponse
			err = json.Unmarshal([]byte(resp.Content), &parsedResp)
			if err != nil {
				fmt.Printf("\n🤖 AI: %s\n", resp.Content)
				return nil
			}

			fmt.Printf("\n🤔 [AI 思考过程]: %s\n", parsedResp.Thought)
			fmt.Printf("🤖 AI: %s\n", parsedResp.Response)

			state.AddTrace("LLM_Thinking", "分析与回答", parsedResp.Thought, parsedResp.Response)
			
			// 重点：在这里我们不直接操作外部的 shortMem 变量了，
			// 而是把 AI 的回复塞进 state 里，让后续节点去处理（保持状态纯粹洁净）
			state.SetData("ai_response", parsedResp.Response)
			return nil
		},
	}

	// 节点 C: 反思与持久化
	nodeReflection := &graph.TreeNode{
		ID:   "node_reflect_03",
		Name: "Reflection_And_Save",
		RunFunc: func(ctx context.Context, state *schema.State) error {
			userInput, _ := state.GetData("user_input")
			aiResponse, ok := state.GetData("ai_response")
			
			if ok {
				// 更新短期记忆
				shortMem.AddMessage(schema.Message{Role: schema.RoleUser, Content: userInput.(string)})
				shortMem.AddMessage(schema.Message{Role: schema.RoleAssistant, Content: aiResponse.(string)})
			}

			inputStr := userInput.(string)
			if strings.Contains(inputStr, "记住") {
				f, err := os.OpenFile(dbPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
				if err != nil {
					return err
				}
				defer f.Close()
				extractInfo := strings.Replace(inputStr, "记住", "", -1)
				f.WriteString("- " + extractInfo + "\n")
				fmt.Println("   [后台系统] 💾 已将重要信息写入长期记忆库！")
			}
			return nil
		},
	}

	// ==========================================
	// 组装蓝图树！(组装的艺术)
	// ==========================================
	blueprintTree := graph.NewTree(nodeLoadMemory) // 根节点是 LoadMemory
	
	// LoadMemory 执行完后，把结果交给 LLMThinking
	blueprintTree.AddChild("node_root_01", nodeLLMThinking)
	
	// LLMThinking 执行完后，把结果交给 Reflection
	blueprintTree.AddChild("node_llm_02", nodeReflection)
	// ==========================================
	// 3. 启动交互循环，每次用户输入都跑一遍 Graph
	// ==========================================
	reader := bufio.NewReader(os.Stdin)
	
	// 【Phase 2 新增】：在循环外部创建一个全局的 State，用于贯穿整个对话
	globalState := schema.NewState()

	for {
		fmt.Print("\n🧑 你: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input)

		if input == "exit" {
			break
		}
		if input == "" {
			continue
		}

		// 【Phase 2 新增】：拦截 /trace 命令
		if input == "/trace" {
			globalState.PrintTraces()
			continue
		}

		// 每次运行 Graph 前，清空上一轮的轨迹，但保留数据
		globalState.Traces = make([]schema.TraceLog, 0)
		
		// 把用户最新的输入放进去
		globalState.SetData("user_input", input)

		// 运行整个树！(传入上下文、起始节点、初始状态)
		err := blueprintTree.Run(context.Background(), blueprintTree.Root, globalState)
		if err != nil {
			log.Printf("Blueprint Tree 运行出错: %v\n", err)
		}
	}
}


