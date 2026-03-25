package main

import (
	"bufio"
	"context"
	"fmt"
	"log"
	"os"
	"strings"

	"github.com/AlannerFranklin/synapse/graph/parser"
	"github.com/AlannerFranklin/synapse/llm"
	"github.com/AlannerFranklin/synapse/schema"
	"github.com/AlannerFranklin/synapse/web"
)

func main() {
	// ==========================================
	// 【新增】启动 WebUI 服务器
	// 注意：StartServer 会阻塞当前进程，所以它后面的 CLI 循环不会执行了。
	// 如果你想同时保留 CLI 和 Web，可以使用 go web.StartServer("8080")
	// 但为了简单，我们可以直接在这里启动 Web。
	// ==========================================
	web.StartServer("8080")
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
	dbPath := "long_term_memory.txt" // 长期记忆文件路径

	// 2. 使用 DSL 解析器，从 JSON 文件动态构建蓝图树！
	// 我们不再在 main.go 里硬编码组装节点了，而是做到“配置即代码”。
	// 注意：shortMem 已经被我们下放到 State 里面了，所以这里不需要传 shortMem 啦！
	blueprintTree, err := parser.ParseFromFile("blueprint.json", provider, dbPath)
	if err != nil {
		log.Fatalf("❌ 加载蓝图配置文件失败: %v", err)
	}

	// ==========================================
	// 3. 启动交互循环，每次用户输入都跑一遍 Graph
	// ==========================================
	reader := bufio.NewReader(os.Stdin)
	
	// 【Phase 2 新增】：在循环外部创建一个全局的 State，用于贯穿整个对话
	globalState := schema.NewState()

	// 【Phase 3 新增】：时光机！保存每一轮对话前的全局状态快照
	var stateHistory []*schema.State

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

		// 【Phase 3 新增】：拦截 /revert 命令 (时光回溯)
		if input == "/revert" {
			if len(stateHistory) > 1 {
				// 回退到上一次对话前的状态 (去掉最后一个快照)
				stateHistory = stateHistory[:len(stateHistory)-1]
				// 恢复全局状态为历史快照的深拷贝
				globalState = stateHistory[len(stateHistory)-1].Clone()
				fmt.Println("⏪ [时光机]: 已成功回溯到上一轮对话状态！你可以重新提问了。")
			} else {
				fmt.Println("❌ [时光机]: 已经没有更早的记忆可以回溯了。")
			}
			continue
		}

		// 每次正常对话前，把当前状态 Clone 一份存入时光机
		stateHistory = append(stateHistory, globalState.Clone())

		// 每次运行 Graph 前，清空上一轮的轨迹，但保留数据和记忆
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


