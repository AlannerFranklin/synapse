package llm

import (
	"context"

	"github.com/AlannerFranklin/synapse/schema" // 导入我们昨天写的 schema 包
)

// ==========================================
// 语法教学：Go 的 context (上下文)
// ==========================================
// context.Context 是 Go 并发编程中最常用的工具。
// 它的主要作用是：控制超时、取消任务、传递请求级别的数据。
// 比如：你向大模型发请求，如果用户突然关掉网页，或者请求超过了30秒，
// 带有 context 的函数就能立刻感知到并停止执行，从而节省资源。
// 惯例：context 永远作为函数的第一个参数，并且变量名通常叫 ctx。

// GenerateOptions 定义了调用大模型时的可选参数
type GenerateOptions struct {
	Temperature float32       // 随机性，0.0 到 2.0 之间
	Tools       []schema.Tool // 告诉大模型当前可用的工具列表
}

// ==========================================
// 语法教学：接口隔离原则
// ==========================================
// Model 接口定义了任何一个大语言模型（如 OpenAI, DeepSeek）都必须具备的能力。
// 我们的框架其他部分（比如 Graph, Memory）只会依赖这个接口，而不会依赖具体的 OpenAI 代码。
// 这样以后你想换模型，只需要写一个新结构体实现这个接口即可，核心框架代码一行都不用改！
type Model interface {
	// Generate 接收一段历史消息，返回大模型的一条新消息
	// 入参：
	//   - ctx: 用于控制超时和取消
	//   - messages: 对话历史（包含 System, User, Assistant 等消息）
	//   - options: 额外参数（如温度、可用工具）
	// 出参：
	//   - schema.Message: 大模型生成的回复（可能是纯文本，也可能是一个 ToolCall）
	//   - error: 如果网络不通或 API Key 错误，这里会返回 error
	Generate(ctx context.Context, messages []schema.Message, options *GenerateOptions) (schema.Message, error)

	// TODO: 未来我们可以增加 GenerateStream 方法来支持打字机流式输出
}