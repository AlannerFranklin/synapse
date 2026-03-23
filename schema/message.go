package schema

// ==========================================
// 语法教学：Go 的自定义类型 (Custom Types)
// ==========================================
// 在 Go 中，我们可以基于内置类型（比如 string, int）创建自己的新类型。
// 这里的语法是：type [新类型名称] [基础类型]
// 为什么这样做？
// 1. 增加代码可读性：看到 Role 就知道这是角色，而不是随便什么字符串。
// 2. 增加类型安全：函数如果要求传入 Role 类型，你就不能随便传一个普通的 string 进去。
type Role string

// ==========================================
// 语法教学：Go 的常量 (Constants)
// ==========================================
// const 用于定义不会改变的值。
// 使用括号 () 可以将多个常量组合在一起定义，让代码更整洁。
const (
	// 定义了四种大模型对话中标准的角色
	RoleSystem    Role = "system"    // 系统人设，比如 "你是一个有用的助手"
	RoleUser      Role = "user"      // 用户输入，比如 "今天天气怎么样"
	RoleAssistant Role = "assistant" // AI 回复的内容
	RoleTool      Role = "tool"      // 工具执行后返回给 AI 的结果
)

// ==========================================
// 语法教学：Go 的结构体 (Struct) 和 标签 (Tags)
// ==========================================
// Struct（结构体）是 Go 中最重要的数据结构，用来把多个相关的数据打包在一起。
// 语法：type [结构体名称] struct { [字段名] [字段类型] }
//
// ⚠️ 重要规则：首字母大小写决定了可见性（Public/Private）
// - 首字母大写（如 ID, Name）：它是公开的，其他包（文件夹）的代码也能访问它。
// - 首字母小写（如 id, name）：它是私有的，只能在当前 schema 包内部使用。
//
// 🏷️ 什么是 Struct Tag（反引号里的内容）？
// - 比如 `json:"id"`
// - 因为我们的程序要和 OpenAI 等大模型通信，通信格式是 JSON。
// - 这个标签是在告诉 Go 自带的 JSON 转换工具："当我把这个结构体变成 JSON 文本时，请把 Go 里面的 'ID' 变成 JSON 里面的 'id'（小写）。"

// ToolCall 表示 LLM 决定调用某个工具时发出的请求
type ToolCall struct {
	ID        string `json:"id"`        // 工具调用的唯一标识符（由大模型生成，如 call_123abc）
	Name      string `json:"name"`      // 要调用的工具名称（如 "get_weather"）
	Arguments string `json:"arguments"` // 传给工具的参数，通常是一个 JSON 格式的字符串（如 '{"city":"Beijing"}'）
}

// ==========================================
// 语法教学：切片 (Slice) 与 omitempty
// ==========================================

// Message 表示对话流中的一条完整消息
type Message struct {
	Role    Role   `json:"role"`    // 这条消息是谁发的（user/assistant/system/tool）
	Content string `json:"content"` // 消息的具体文本内容

	// 什么是 Slice（切片）？
	// - Go 里面的 []ToolCall 就是切片，相当于 Python 的 List 或者 Java 的 ArrayList。
	// - 它是一个可以动态增加长度的数组。
	//
	// 什么是 omitempty？
	// - 在 json tag 里加上 ,omitempty 意思是："如果这个切片是空的（或者为 nil），在转成 JSON 的时候，直接忽略这个字段，不要输出 'tool_calls': [] 这种多余的东西"。
	ToolCalls []ToolCall `json:"tool_calls,omitempty"` // 如果是 Assistant 想调工具，这个切片里就会有数据

	// ToolCallID 只有当 Role 是 "tool" 时才需要填。
	// 作用是告诉大模型："这是你之前发起的那个 ID 为 xxx 的工具调用，现在我把执行结果还给你了"。
	ToolCallID string `json:"tool_call_id,omitempty"`
}

