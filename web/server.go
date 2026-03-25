package web

import (
	"context"
	"encoding/json"
	"fmt"
	"io"
	"log"
	"net/http"
	"os"
	"sync"
	"time"

	"github.com/AlannerFranklin/synapse/graph/parser"
	"github.com/AlannerFranklin/synapse/llm"
	"github.com/AlannerFranklin/synapse/schema"
)

// ChatNode 代表对话树上的一个节点（一次用户输入及AI回复）
type ChatNode struct {
	ID        string            `json:"id"`
	ParentID  string            `json:"parent_id"`
	UserInput string            `json:"user_input"`
	Reply     string            `json:"reply"`
	Traces    []schema.TraceLog `json:"traces"`
	Timestamp int64             `json:"timestamp"`
	State     *schema.State     `json:"-"` // 这一轮执行完毕后的全局状态快照
}

var (
	chatTreeMu sync.RWMutex
	chatTree   = make(map[string]*ChatNode) // 保存所有的对话节点
	currentID  string                       // 当前聚焦的节点 ID
	rootID     = "root_node"                // 统一的虚拟根节点 ID
)

// init 函数在包加载时执行，用来初始化那个唯一的虚拟根节点
func init() {
	chatTree[rootID] = &ChatNode{
		ID:        rootID,
		ParentID:  "",
		UserInput: "System Initialization",
		Reply:     "你好！我是 Synapse Agent。请先在左侧配置好你的 API Key，然后我们可以开始聊天！",
		Traces:    []schema.TraceLog{},
		Timestamp: time.Now().UnixMilli(),
		State:     schema.NewState(), // 初始的干净状态
	}
	currentID = rootID
}

// ChatRequest 表示前端发来的请求格式
type ChatRequest struct {
	Message   string `json:"message"`
	ApiKey    string `json:"api_key"`
	ApiBase   string `json:"api_base"` // 新增：API Base URL
	ApiType   string `json:"api_type"` // 新增：API 类型，比如 deepseek, openai, qwen 等
	Blueprint string `json:"blueprint"`
	ParentID  string `json:"parent_id"` // 指定从哪个历史节点分叉
}

// ChatResponse 表示返回给前端的响应格式
type ChatResponse struct {
	ID       string            `json:"id"`
	ParentID string            `json:"parent_id"`
	Reply    string            `json:"reply"`
	Traces   []schema.TraceLog `json:"traces"`
	Error    string            `json:"error,omitempty"`
}

// StartServer 启动 WebUI 服务器
func StartServer(port string) {
	// 1. 静态文件服务：将 web/static 目录映射到根路径 /
	http.Handle("/", http.FileServer(http.Dir("web/static")))

	// 2. 核心聊天接口
	http.HandleFunc("/api/chat", handleChat)
	http.HandleFunc("/api/tree", handleGetTree)
	http.HandleFunc("/api/blueprint", handleGetBlueprint) // 新增导出蓝图接口

	fmt.Printf("========================================\n")
	fmt.Printf("🌐 Synapse WebUI 已启动！\n")
	fmt.Printf("👉 请在浏览器打开: http://localhost:%s\n", port)
	fmt.Printf("========================================\n")

	if err := http.ListenAndServe(":"+port, nil); err != nil {
		log.Fatalf("Web 服务器启动失败: %v", err)
	}
}

func handleGetTree(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	chatTreeMu.RLock()
	defer chatTreeMu.RUnlock()

	// 把 map 转换成 slice 返回
	nodes := make([]*ChatNode, 0, len(chatTree))
	for _, node := range chatTree {
		nodes = append(nodes, node)
	}
	
	json.NewEncoder(w).Encode(map[string]interface{}{
		"nodes":     nodes,
		"currentId": currentID,
	})
}

func handleGetBlueprint(w http.ResponseWriter, r *http.Request) {
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	// 优先尝试读取前端可能上传过的临时蓝图文件
	data, err := os.ReadFile("web_temp_blueprint.json")
	if err != nil {
		// 如果没有临时文件，读取默认蓝图
		data, err = os.ReadFile("blueprint.json")
		if err != nil {
			http.Error(w, `{"error":"找不到蓝图文件"}`, http.StatusNotFound)
			return
		}
	}
	
	w.Write(data)
}

func handleChat(w http.ResponseWriter, r *http.Request) {
	// 允许跨域（如果有需要）
	w.Header().Set("Access-Control-Allow-Origin", "*")
	w.Header().Set("Content-Type", "application/json")

	if r.Method != http.MethodPost {
		http.Error(w, "只支持 POST 请求", http.StatusMethodNotAllowed)
		return
	}

	// 1. 解析请求体
	body, _ := io.ReadAll(r.Body)
	var req ChatRequest
	if err := json.Unmarshal(body, &req); err != nil {
		json.NewEncoder(w).Encode(ChatResponse{Error: "请求参数解析失败"})
		return
	}

	if req.ApiKey == "" {
		json.NewEncoder(w).Encode(ChatResponse{Error: "请在左侧设置中输入 API Key"})
		return
	}

	// 2. 初始化大模型 Provider
	// 如果前端传了 ApiBase，优先使用前端的
	apiBase := req.ApiBase
	if apiBase == "" {
		// 根据常见的提供商设置默认 base
		switch req.ApiType {
		case "openai":
			apiBase = "https://api.openai.com/v1"
		case "qwen":
			apiBase = "https://dashscope.aliyuncs.com/compatible-mode/v1"
		default:
			apiBase = "https://api.deepseek.com/v1" // 默认 deepseek
		}
	}
	
	// 默认使用一个通用模型名字，实际在真实的 provider 中可以更细化，这里先简单传给 NewOpenAIProvider
	// 因为大部分都是兼容 OpenAI 接口的
	modelName := "deepseek-chat"
	if req.ApiType == "openai" {
		modelName = "gpt-3.5-turbo"
	} else if req.ApiType == "qwen" {
		modelName = "qwen-plus"
	}

	provider := llm.NewOpenAIProvider(apiBase, req.ApiKey, modelName)

	// 3. 处理蓝图文件
	bpPath := "blueprint.json"
	// 如果用户在前端修改了蓝图，我们就临时存一个文件供 parser 使用
	if req.Blueprint != "" {
		bpPath = "web_temp_blueprint.json"
		os.WriteFile(bpPath, []byte(req.Blueprint), 0644)
	}

	// 4. 解析图结构
	tree, err := parser.ParseFromFile(bpPath, provider, "long_term_memory.txt")
	if err != nil {
		json.NewEncoder(w).Encode(ChatResponse{Error: "蓝图解析失败: " + err.Error()})
		return
	}

	// 5. 准备状态机：根据 ParentID 进行分叉
	chatTreeMu.Lock()
	var currentState *schema.State
	targetParentID := req.ParentID
	
	// 如果前端没有传 parent_id，说明它想在当前的 currentID 后面继续说
	if targetParentID == "" {
		targetParentID = currentID
	}

	// 尝试找到父节点（现在一定能找到，至少有 root_node）
	parentNode, exists := chatTree[targetParentID]
	if exists {
		// 从父节点状态深度拷贝（记忆分叉核心！）
		currentState = parentNode.State.Clone()
	} else {
		// 防御性编程：如果真的没找到，退化到 root_node
		currentState = chatTree[rootID].State.Clone()
		targetParentID = rootID
	}
	chatTreeMu.Unlock()

	// 每次对话清空上一轮的轨迹，但保留历史消息和数据
	currentState.Traces = make([]schema.TraceLog, 0)
	currentState.SetData("user_input", req.Message)

	// 6. 运行蓝图树！
	err = tree.Run(context.Background(), tree.Root, currentState)
	
	// 7. 构建响应并保存到历史树
	newID := fmt.Sprintf("node_%d", time.Now().UnixNano())
	resp := ChatResponse{
		ID:       newID,
		ParentID: targetParentID,
		Traces:   currentState.Traces,
	}

	if err != nil {
		resp.Error = "蓝图运行失败: " + err.Error()
	} else {
		if finalResp, ok := currentState.GetData("ai_response"); ok {
			resp.Reply = finalResp.(string)
		} else {
			resp.Reply = "节点执行完毕，但未在状态机中找到 ai_response 结果。"
		}
	}

	// 保存节点
	chatTreeMu.Lock()
	newNode := &ChatNode{
		ID:        newID,
		ParentID:  targetParentID,
		UserInput: req.Message,
		Reply:     resp.Reply,
		Traces:    resp.Traces,
		Timestamp: time.Now().UnixMilli(),
		State:     currentState.Clone(), // 存一份执行完后的状态快照
	}
	chatTree[newID] = newNode
	currentID = newID // 更新当前指向
	chatTreeMu.Unlock()

	// 8. 返回结果给前端
	json.NewEncoder(w).Encode(resp)
}