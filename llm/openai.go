package llm

import (
	"bytes"
	"context"
	"encoding/json"
	"fmt"
	"io"
	"net/http"

	"github.com/AlannerFranklin/synapse/schema"
)

// ==========================================
// 1. 定义 OpenAI 兼容的 HTTP 请求和响应结构体
// ==========================================
// 这些结构体完全按照 OpenAI 官方 API 文档来定义的。
// 我们在字段后面加上 json tag，Go 就能自动帮我们把它们转成 JSON。

// chatRequest 是我们发给大模型的 JSON 数据结构

type chatRequest struct {
	Model       string           `json:"model"`
	Messages    []schema.Message `json:"messages"`
	Temperature float32          `json:"temperature,omitempty"`
	// TODO: 工具调用的结构体比较复杂，我们先留空，等把基础对话跑通了再加
}

// chatResponse 是大模型返回给我们的 JSON 数据结构

type chatResponse struct {
	Choices []struct {
		Message schema.Message `json:"message"`
	} `json:"choices"`
	// 我们目前只关心返回的 Message，其他字段（如 token 消耗量）先忽略
}

// ==========================================
// 2. 定义 OpenAIProvider 结构体
// ==========================================
// 这个结构体实现了我们之前在 llm.go 里定义的 Model 接口
type OpenAIProvider struct {
	baseURL string
	apiKey  string
	model   string // 比如 "deepseek-chat" 或 "qwen2.5:7b"
	client  *http.Client
}

// ==========================================
// 3. 编写构造函数
// ==========================================
// NewOpenAIProvider 创建一个新的大模型客户端
//
// 💡 这里就是实现“一码多用”的秘密：
// - 如果用 DeepSeek，baseURL 传 "https://api.deepseek.com/v1"，apiKey 传你的 sk-xxx
// - 如果用 Ollama，baseURL 传 "http://localhost:11434/v1"，apiKey 传 "" (随便传)
// - 如果用 OpenAI，baseURL 传 "https://api.openai.com/v1"，apiKey 传 sk-xxx

func NewOpenAIProvider(baseURL, apiKey, model string)*OpenAIProvider {
	return &OpenAIProvider{
		baseURL: baseURL,
		apiKey: apiKey,
		model: model,
		client: &http.Client{}, // 使用 Go 自带的默认 HTTP 客户端
	}
}

// 确保 OpenAIProvider 实现了 Model 接口（这是一个编译期检查的小技巧）
var _ Model = (*OpenAIProvider)(nil)

// ==========================================
// 4. 实现 Generate 方法 (核心网络请求逻辑)
// ==========================================
//
// 语法教学：方法的接收者 (Receiver)
// `func (p *OpenAIProvider) Generate(...)` 里的 `(p *OpenAIProvider)` 就是接收者。
// 这意味着 Generate 是属于 OpenAIProvider 结构体的一个“方法”。你可以通过 `provider.Generate()` 来调用它。
//
// 语法教学：多返回值 (Multiple Return Values)
// Go 的函数可以返回多个值。这里返回了 `(schema.Message, error)`。
// 这是一种非常经典的 Go 错误处理模式：第一个值是正常的返回结果，第二个值是可能发生的错误。
//
func (p *OpenAIProvider) Generate(ctx context.Context, messages []schema.Message, options *GenerateOptions) (schema.Message, error) {
	// ==========================================
	// 1. 组装请求数据 (Struct Initialization)
	// ==========================================
	// 这里的 `chatRequest{}` 是在初始化我们在上面定义的结构体。
	// 注意 Go 语言里的逗号：在多行初始化时，每一行的末尾（包括最后一行）都必须有逗号 `,`。
	reqData := chatRequest{
		Model:    p.model,
		Messages: messages,
	}
	
	// 语法教学：指针的判空
	// options 是一个指针 (*GenerateOptions)。在 Go 里，如果调用者没有传这个参数（传了 nil），
	// 我们直接去访问 options.Temperature 会导致程序崩溃（Panic：空指针异常）。
	// 所以在使用指针前，必须先判断它是不是 nil。
	if options != nil {
		reqData.Temperature = options.Temperature
	}

	// ==========================================
	// 2. 将 Go 结构体转换为 JSON 字节流 (Marshal)
	// ==========================================
	// json.Marshal 是 Go 标准库 `encoding/json` 提供的方法。
	// 它的作用是把 Go 内存里的结构体（reqData），变成一段 JSON 格式的字节数组（[]byte）。
	// 
	// 语法教学：错误处理惯用法 (Error Handling)
	// `if err != nil { ... }` 是 Go 语言里写得最多的一句话。
	// Go 没有 try...catch，当调用的函数可能出错时，我们必须立刻检查 err 是否为 nil。
	reqBytes, err := json.Marshal(reqData)
	if err != nil {
		// fmt.Errorf 用来格式化错误信息。
		// `%w` 是一个特殊的占位符（Wrap），它会把底层的 err 包装进新的错误信息里，方便日后排查问题。
		// 因为出错了，所以我们返回一个空的 schema.Message{}，以及这个错误。
		return schema.Message{}, fmt.Errorf("failed to marshal request: %w", err)
	}

	// ==========================================
	// 3. 构建 HTTP 请求对象 (http.NewRequest)
	// ==========================================
	url := p.baseURL + "/chat/completions"
	
	// http.NewRequestWithContext 是创建一个 HTTP 请求对象的标准做法。
	// 参数1：ctx (上下文，用于控制超时和取消)
	// 参数2："POST" (请求方法)
	// 参数3：url (请求地址)
	// 参数4：请求体 (Body)。
	// 注意：HTTP 请求的 Body 必须是一个实现了 `io.Reader` 接口的流。
	// 我们的 reqBytes 是一个普通的字节数组 `[]byte`，不能直接传进去。
	// 所以我们用 `bytes.NewReader(reqBytes)` 把数组包装成了一个可以被读取的流。
	req, err := http.NewRequestWithContext(ctx, "POST", url, bytes.NewReader(reqBytes))
	if err != nil {
		return schema.Message{}, fmt.Errorf("failed to create request: %w", err)
	}

	// ==========================================
	// 4. 设置 HTTP 请求头 (Headers)
	// ==========================================
	// 告诉大模型的服务器：我发给你的数据是 JSON 格式的。
	req.Header.Set("Content-Type", "application/json")
	
	// 鉴权：只有当 apiKey 不为空时，才设置 Authorization 头。
	// 比如本地的 Ollama 模型通常不需要密码，而 DeepSeek/OpenAI 必须要有。
	if p.apiKey != "" {
		req.Header.Set("Authorization", "Bearer "+p.apiKey)
	}

	// ==========================================
	// 5. 真正发送 HTTP 请求 (Client.Do)
	// ==========================================
	// p.client.Do(req) 会把我们刚刚捏好的请求发到网络上，并等待服务器响应。
	// 这是一个阻塞操作（会卡在这里等），直到拿到结果或者超时。
	resp, err := p.client.Do(req)
	if err != nil {
		return schema.Message{}, fmt.Errorf("http request failed: %w", err)
	}
	
	// ==========================================
	// 语法教学：defer 延迟执行（极其重要！）
	// ==========================================
	// `defer` 关键字的意思是：“把这行代码推迟到当前函数（Generate）即将结束的那一瞬间再去执行”。
	// 为什么要这样？
	// 因为网络请求返回的 resp.Body 占用着系统资源（比如 TCP 连接）。
	// 如果你忘了关闭它，就会导致“内存泄漏”。
	// 写上 defer，无论下面的代码是正常结束还是报错 return 了，Go 都会保证帮你执行 Close()。
	defer resp.Body.Close()

	// ==========================================
	// 6. 检查 HTTP 状态码 (Status Code)
	// ==========================================
	// HTTP 200 (http.StatusOK) 代表请求成功。
	if resp.StatusCode != http.StatusOK {
		// 如果状态码不是 200，说明大模型那边报错了（比如你的 API Key 填错了，或者欠费了）。
		// 我们用 io.ReadAll 把错误信息从流里全部读出来，转成字符串返回给用户看。
		bodyErr, _ := io.ReadAll(resp.Body)
		return schema.Message{}, fmt.Errorf("api error (status %d): %s", resp.StatusCode, string(bodyErr))
	}

	// ==========================================
	// 7. 解析返回的 JSON 响应 (Decode)
	// ==========================================
	// 声明一个空的 chatResponse 结构体准备接收数据。
	var respData chatResponse
	
	// json.NewDecoder(流).Decode(&变量指针) 是解析 JSON 流的标准写法。
	// 为什么要传 `&respData`（取地址符）？
	// 如果你只传 `respData`，Go 会把结构体复制一份传给 Decode 函数，Decode 往里面塞满数据后，
	// 你外面的 `respData` 依然是空的。传指针 `&` 才能让 Decode 直接修改你外面的变量。
	if err := json.NewDecoder(resp.Body).Decode(&respData); err != nil {
		return schema.Message{}, fmt.Errorf("failed to decode response: %w", err)
	}

	// ==========================================
	// 8. 提取大模型的回复内容
	// ==========================================
	// len() 函数用来获取切片 (Slice) 的长度。
	// 如果大模型什么都没返回（虽然很少见），我们要防范数组越界错误。
	if len(respData.Choices) == 0 {
		return schema.Message{}, fmt.Errorf("empty response choices")
	}

	// 返回第一条选择 (Choices[0]) 里面的 Message。
	// 此时 error 返回 nil，代表一切顺利！
	return respData.Choices[0].Message, nil
}