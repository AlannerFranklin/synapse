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
/*
func main() {
	// ==========================================
	// Day 4 测试：图执行引擎 (Graph) 的威力
	// 我们来模拟一个场景：你想买一台手机。
	// 1. (串行) 节点A：确定你想买什么品牌的手机。
	// 2. (并行) 节点B：同时去淘宝、京东、拼多多查价格。
	// 3. (串行) 节点C：汇总价格，给出购买建议。
	// ==========================================

	// 1. 创建全局状态
	state := schema.NewState()

	// 2. 创建图
	g := graph.NewGraph()

	// ------------------------------------------
	// 节点A：确定需求（串行）
	// ------------------------------------------
	nodeA := graph.NewNode("确定需求", func(ctx context.Context, s *schema.State) error {
		fmt.Println("   [确定需求] 用户想买: iPhone 16 Pro")
		// 把数据写入全局状态，供后面的节点使用
		s.SetData("target_phone", "iPhone 16 Pro")
		time.Sleep(500 * time.Millisecond) // 模拟思考时间
		return nil
	})

	// ------------------------------------------
	// 节点B：全网比价（并行！）
	// ------------------------------------------
	// 注意这里：我们把 3 个查询函数放进了一个并行节点里。
	// 它们会同时启动，谁也不等谁！
	nodeB := graph.NewParallelNode("全网比价",
		// 任务1：查淘宝
		func(ctx context.Context, s *schema.State) error {
			phone, _ := s.GetData("target_phone")
			fmt.Printf("   [查淘宝] 正在搜索 %s...\n", phone)
			time.Sleep(2 * time.Second) // 模拟网络延迟 2 秒
			fmt.Println("   [查淘宝] 搜索完毕：7999 元")
			s.SetData("price_taobao", 7999)
			return nil
		},
		// 任务2：查京东
		func(ctx context.Context, s *schema.State) error {
			phone, _ := s.GetData("target_phone")
			fmt.Printf("   [查京东] 正在搜索 %s...\n", phone)
			time.Sleep(1 * time.Second) // 模拟网络延迟 1 秒 (京东比较快)
			fmt.Println("   [查京东] 搜索完毕：8099 元")
			s.SetData("price_jd", 8099)
			return nil
		},
		// 任务3：查拼多多
		func(ctx context.Context, s *schema.State) error {
			phone, _ := s.GetData("target_phone")
			fmt.Printf("   [查拼多多] 正在搜索 %s...\n", phone)
			time.Sleep(3 * time.Second) // 模拟网络延迟 3 秒 (拼多多最慢)
			fmt.Println("   [查拼多多] 搜索完毕：7599 元 (百亿补贴!)")
			s.SetData("price_pdd", 7599)
			return nil
		},
	)

	// ------------------------------------------
	// 节点C：汇总建议（串行）
	// ------------------------------------------
	nodeC := graph.NewNode("汇总建议", func(ctx context.Context, s *schema.State) error {
		tb, _ := s.GetData("price_taobao")
		jd, _ := s.GetData("price_jd")
		pdd, _ := s.GetData("price_pdd")

		fmt.Println("   [汇总建议] 价格对比结果：")
		fmt.Printf("      - 淘宝: %v\n", tb)
		fmt.Printf("      - 京东: %v\n", jd)
		fmt.Printf("      - 拼多多: %v\n", pdd)
		fmt.Println("   👉 结论：建议去拼多多买，最便宜！")
		return nil
	})

	// 3. 把节点按顺序加入到图中
	g.AddNode(nodeA)
	g.AddNode(nodeB)
	g.AddNode(nodeC)

	// 4. 运行整个图，并计时
	fmt.Println("🚀 开始执行 Agent 工作流...")
	startTime := time.Now()

	err := g.Run(context.Background(), state)
	if err != nil {
		log.Fatalf("图执行失败: %v", err)
	}

	// 算一下总共花了多少时间
	fmt.Printf("✅ 执行完毕！总耗时: %v\n", time.Since(startTime))
	// 思考题：
	// 淘宝要2秒，京东要1秒，拼多多要3秒。
	// 如果是串行，总共需要 2+1+3 = 6秒。
	// 既然我们是并行，你猜总耗时会是多少秒？
}
*/
/*
func main() {
	// ==========================================
	// 1. 初始化 LLM 客户端
	// ==========================================

	apiKeyBytes, err := os.ReadFile("api.txt")
	if err != nil {
		log.Fatalf("无法读取 api.txt 文件: %v", err)
	}
	apiKey := strings.TrimSpace(string(apiKeyBytes))
	provider := llm.NewOpenAIProvider(
		"https://api.deepseek.com/v1",
		apiKey,
		"deepseek-chat",
	)
	// ==========================================
	// 2. 初始化短期记忆 (滑动窗口)
	// 我们设置 maxSize = 4，意味着它最多只能记住最近的 4 句话（2轮对话）
	// ==========================================
	mem := memory.NewShortTermMemory(4)
	// 先在记忆里塞入一条系统提示，定下人设
	mem.AddMessage(schema.Message{
		Role:    schema.RoleSystem,
		Content: "你是一个只能用一句简短的话回答问题的助手。",
	})

	fmt.Println("🤖 AI 已启动！(输入 'exit' 退出)")
	fmt.Println("--------------------------------")

	// ==========================================
	// 3. 启动交互式命令行循环
	// ==========================================
	// bufio.NewReader 用于读取用户在终端里的键盘输入
	reader := bufio.NewReader(os.Stdin)

	for {
		// 打印提示符，等待用户输入
		fmt.Print("🧑 你: ")
		input, _ := reader.ReadString('\n')
		input = strings.TrimSpace(input) // 去掉末尾的回车换行符

		// 如果输入 exit，退出循环
		if input == "exit" {
			fmt.Println("👋 再见！")
			break
		}
		if input == "" {
			continue
		}

		// ------------------------------------------
		// 步骤 A: 把用户说的话存入记忆
		// ------------------------------------------
		mem.AddMessage(schema.Message {
			Role:    schema.RoleUser,
			Content: input,
		})
		// ------------------------------------------
		// 步骤 B: 取出当前的全部记忆，发给大模型
		// ------------------------------------------
		history := mem.GetMessages()
		// 打印一下当前发送了多少条消息，让你直观看到滑动窗口的效果
		fmt.Printf("   [系统提示: 当前发送给 AI 的上下文消息数量为: %d]\n", len(history))

		respMsg, err := provider.Generate(context.Background(), history, &llm.GenerateOptions{
			Temperature: 0.7,
		})
		if err != nil {
			log.Printf("❌ 调用大模型失败: %v\n", err)
			continue
		}

		// ------------------------------------------
		// 步骤 C: 打印 AI 的回复，并把它也存入记忆！
		// (如果不存，AI 就不知道自己刚才说了什么)
		// ------------------------------------------
		fmt.Printf("🤖 AI: %s\n\n", respMsg.Content)
		mem.AddMessage(respMsg)
	}
}
*/

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

	// 2. 构建 Agent 执行图 (DAG)
	agentGraph := graph.NewGraph()
	// ==========================================
	// 节点 A: 提取长期记忆
	// ==========================================
	agentGraph.AddNode(graph.NewNode("LoadLongTerMemory", func(ctx context.Context, state *schema.State) error {
		// 从文件里读取长期记忆（如果文件不存在则忽略）
		content, err := os.ReadFile(dbPath)
		var longTermStr string
		if err == nil && len(content) > 0 {
			longTermStr = string(content)
		} else {
			longTermStr = "暂无。"
		}
		// 放到全局 State 里，供下一个节点使用
		state.SetData("long_term_knowledge", longTermStr)
		return nil
	}))

	// ==========================================
	// 节点 B: 大模型核心思考
	// ==========================================
	/*agentGraph.AddNode(graph.NewNode("LLM_Thinking", func(ctx context.Context, state *schema.State) error {
		// 取出用户当前的输入
		userInput, _ := state.GetData("user_input")
		longTermKnowledge, _ := state.GetData("long_term_knowledge")

// 构建系统提示词，把长期记忆塞进去！
		sysPrompt := fmt.Sprintf(`你是一个聪明的 AI 助手。
这是你对这个用户的长期记忆（如果你记住了他的名字或喜好，请利用这些信息）：
[%s]

请简短、幽默地回答用户的问题。`, longTermKnowledge)
		// 组装最终的对话流：系统提示 + 短期记忆记录 + 用户最新输入
		finalMessages := []schema.Message{{Role: schema.RoleSystem, Content:sysPrompt}}
		finalMessages = append(finalMessages, shortMem.GetMessages()...) //加上短期记忆
		finalMessages = append(finalMessages, schema.Message{Role: schema.RoleUser, Content: userInput.(string)})

		// 调用大模型
		resp, err := provider.Generate(ctx, finalMessages, &llm.GenerateOptions{Temperature: 0.7})
		if err != nil {
			return err
		}
		// 把 AI 的回复存入短期记忆，并放入 State
		shortMem.AddMessage(schema.Message{Role: schema.RoleUser, Content: userInput.(string)})
		shortMem.AddMessage(resp)
		state.SetData("ai_response", resp.Content)

		fmt.Printf("\n🤖 AI: %s\n", resp.Content)
		return nil
	}))*/
	// ==========================================
	// 节点 B: 大模型核心思考 (带 JSON 思考链捕获)
	// ==========================================
	agentGraph.AddNode(graph.NewNode("LLM_Thinking", func(ctx context.Context, state *schema.State) error {
		userInput, _ := state.GetData("user_input")
		longTermKnowledge, _ := state.GetData("long_term_knowledge")

		// 1. 改造系统提示词，强迫它输出 JSON
		sysPrompt := fmt.Sprintf(`你是一个聪明的 AI 助手。
这是你对用户的长期记忆：[%s]

【强制要求】
你必须且只能返回一个 JSON 格式的字符串，不能包含任何其他多余的文字或 Markdown 代码块标记（不要输出 json 标记）。
JSON 的结构必须如下：
{
  "thought": "这里写你一步步的思考过程，你是怎么想的",
  "response": "这里写你最终要回答给用户的话"
}`, longTermKnowledge)

		finalMessages := []schema.Message{{Role: schema.RoleSystem, Content: sysPrompt}}
		finalMessages = append(finalMessages, shortMem.GetMessages()...)
		finalMessages = append(finalMessages, schema.Message{Role: schema.RoleUser, Content: userInput.(string)})

		// 2. 调用大模型
		resp, err := provider.Generate(ctx, finalMessages, &llm.GenerateOptions{Temperature: 0.7})
		if err != nil {
			return err
		}

		// ==========================================
		// 语法教学：Go 语言解析 JSON (反序列化)
		// ==========================================
		// 在 Go 中，我们通常定义一个结构体 (struct) 来接收 JSON 数据。
		// 结构体字段后面的 `json:"xxx"` 叫做 Tag（标签），告诉 Go 的 json 包：
		// "请把 JSON 里的 thought 字段塞到我的 Thought 变量里"。
		type LLMResponse struct {
			Thought  string `json:"thought"`
			Response string `json:"response"`
		}

		var parsedResp LLMResponse
		// json.Unmarshal 接收两个参数：
		// 参数1：要解析的 JSON 字符串的字节数组 ([]byte)
		// 参数2：你要把数据存到哪里的指针 (&parsedResp)
		err = json.Unmarshal([]byte(resp.Content), &parsedResp)
		if err != nil {
			// 如果大模型不听话，没返回标准 JSON，我们就回退到普通模式
			fmt.Printf("\n🤖 AI: %s\n", resp.Content)
			state.AddTrace("LLM_Thinking", "直接回复(非JSON)", "模型未按要求输出JSON", resp.Content)
			
			shortMem.AddMessage(schema.Message{Role: schema.RoleUser, Content: userInput.(string)})
			shortMem.AddMessage(resp)
			state.SetData("ai_response", resp.Content)
			return nil
		}

		// 3. 成功解析！记录 TraceLog 并保存记忆
		fmt.Printf("\n🤔 [AI 思考过程]: %s\n", parsedResp.Thought)
		fmt.Printf("🤖 AI: %s\n", parsedResp.Response)

		// 将大模型的思考过程写入我们刚做的 Trace 记录里！
		state.AddTrace("LLM_Thinking", "分析与回答", parsedResp.Thought, parsedResp.Response)

		// 保存到短期记忆（注意：存给大模型看的只存最终回复，别把思考过程也存进去了，会浪费 Token）
		shortMem.AddMessage(schema.Message{Role: schema.RoleUser, Content: userInput.(string)})
		shortMem.AddMessage(schema.Message{Role: schema.RoleAssistant, Content: parsedResp.Response})
		state.SetData("ai_response", parsedResp.Response)

		return nil
	}))
	// ==========================================
	// 节点 C: 反思与持久化 (后台并行！)
	// ==========================================
	// 我们用并行节点来做这件事，这样用户就不需要等待 AI 慢慢反思了。
	agentGraph.AddNode(graph.NewParallelNode("Reflection_And_Save", func(ctx context.Context, state *schema.State) error {
		userInput, _ := state.GetData("user_input")
		inputStr := userInput.(string)
		// 简单的规则：如果用户的话里包含 "记住"，我们就把它写到长期记忆文件里
		if strings.Contains(inputStr, "记住") {
			// 以追加模式打开文件
			f, err := os.OpenFile(dbPath, os.O_APPEND|os.O_CREATE|os.O_WRONLY, 0644)
			/*
			dbPath：要打开的文件路径（字符串）。

			os.O_APPEND|os.O_CREATE|os.O_WRONLY：通过位或运算（|）组合多个打开模式：

			os.O_APPEND：以追加模式写入，写入的数据会添加到文件末尾。

			os.O_CREATE：如果文件不存在，则创建新文件。

			os.O_WRONLY：以只写模式打开文件（不能读取）。

			0644：文件权限（当文件被创建时使用）。0644 表示：

			所有者（owner）可读、可写（6 = 4+2）

			组用户（group）只读（4）

			其他用户（other）只读（4）
			*/
			if err != nil {
				return err
			}
			defer f.Close()

			// 写入文件
			extractInfo := strings.Replace(inputStr, "记住", "", -1)
			f.WriteString("- " + extractInfo + "\n")
			fmt.Println("   [后台系统] 💾 已将重要信息写入长期记忆库！")
		}
		return nil
	}))
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

		// 运行整个图！
		err := agentGraph.Run(context.Background(), globalState)
		if err != nil {
			log.Printf("Graph 运行出错: %v\n", err)
		}
	}
}


