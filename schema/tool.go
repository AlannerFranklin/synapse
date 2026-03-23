package schema
// ==========================================
// 语法教学：Go 的函数签名 (Function Signature)
// ==========================================
// 这行代码定义了一个“函数类型”。
// 在 Go 中，函数是一等公民，可以像变量一样传来传去。
// 这里的 `ToolFunc` 代表一种特定的函数长相：
// - 它接收一个 string 类型的参数（通常是大模型传过来的 JSON 参数）
// - 它返回两个值：(string, error)。第一个是执行结果，第二个是错误信息。
// 
// 为什么要返回 error？
// Go 语言没有 try-catch。Go 习惯在函数最后返回一个 error 类型。
// 如果没出错，error 就是 nil（空）。
type ToolFunc func(arguments string) (string, error)
// ==========================================
// 语法教学：Go 的接口 (Interface)
// ==========================================
// 接口是 Go 语言中最强大的特性之一！
// 它不是定义一个具体的东西，而是定义“一种行为规范”。
// 只要某个结构体“拥有”下面这三个方法（Name, Description, Execute），
// Go 语言就认为这个结构体是一个 Tool。你不需要显式地写 "implements Tool"。
type Tool interface {
	// Name 返回工具的名称（比如 "get_weather"）
	Name() string

	// Description 返回工具的描述，这段描述是给大模型看的，让大模型知道什么时候该用这个工具
	Description() string

	// Execute 实际执行这个工具。传入参数，返回结果或错误。
	Execute(arguments string) (string, error)
}

// ==========================================
// 补充：一个简单的 Tool 结构体示例（非接口）
// ==========================================
// 虽然接口很灵活，但有时候我们只需要一个简单的结构体来注册工具。
// 我们可以定义一个 BasicTool 结构体来实现上面的 Tool 接口。

type BasicTool struct {
	ToolName        string   // 工具名称
	ToolDescription string   // 工具描述
	Func            ToolFunc // 真正要执行的那个函数
}

// ==========================================
// 语法教学：为结构体添加方法 (Methods)
// ==========================================
// 这不是普通的函数，这是一个“方法”。
// 注意 `(t *BasicTool)` 这个部分，它叫接收者 (Receiver)。
// 它意味着：Name() 这个方法是属于 BasicTool 的。相当于面向对象里的 t.Name()。
// 为什么用 `*BasicTool` (指针) 而不是 `BasicTool`？
// - 用指针可以避免复制整个结构体，效率更高。

func (t *BasicTool) Name() string {
	return t.ToolName
}

func (t *BasicTool) Description() string {
	return t.ToolDescription
}

func (t *BasicTool) Execute(arguments string) (string, error) {
	// 调用我们存放在结构体里的那个函数
	return t.Func(arguments)
}