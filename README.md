# Synapse 🧠

一个主打 **并行执行 (DAG)** 与 **深度记忆 (Memory)** 的轻量级 Go 语言 LLM Agent 框架。

## 🌟 为什么选择 Synapse？

市面上已经有了 LangChain、Eino 等优秀的框架，但它们往往为了兼容性而引入了非常沉重的抽象层。
Synapse 旨在提供一个**极简、原生、高性能**的替代方案：
- **Go 原生并发**：利用 Go 的 `goroutine` 和 `channel`，天生支持多 Agent 节点的并行调度（Fan-out / Fan-in）。
- **零冗余抽象**：不到 1000 行核心代码，看懂源码只需要半天，极度容易二次开发。
- **记忆优先**：内置短期滑动窗口记忆与长期向量记忆接口，让你的 Agent 拥有真正的“潜意识”。
- **多模型兼容**：一套代码，完美兼容 OpenAI、DeepSeek、硅基流动以及本地的 Ollama 模型。

## 🚀 核心特性 (开发中)

- [x] **统一 Schema**：标准化 Message, Tool, State 数据结构
- [x] **LLM Provider**：极简的 OpenAI 兼容客户端 (支持 DeepSeek, Ollama)
- [ ] **DAG 执行引擎**：基于图的串行/并行任务调度器
- [ ] **Memory 模块**：Short-term & Long-term 记忆管理
- [ ] **Tool Calling**：支持本地函数注册与大模型回调

## 🛠️ 快速开始

### 1. 引入模块
```bash
go get github.com/AlannerFranklin/synapse
```

### 2. 基础对话 (DeepSeek 示例)
```go
package main

import (
	"context"
	"fmt"
	"github.com/AlannerFranklin/synapse/llm"
	"github.com/AlannerFranklin/synapse/schema"
)

func main() {
	// 初始化 Provider (兼容 DeepSeek)
	provider := llm.NewOpenAIProvider(
		"https://api.deepseek.com/v1", // BaseURL
		"sk-your-deepseek-api-key",    // API Key
		"deepseek-chat",               // 模型名称
	)

	// 准备消息
	messages := []schema.Message{
		{Role: schema.RoleSystem, Content: "你是一个幽默的程序员鼓励师。"},
		{Role: schema.RoleUser, Content: "我今天写代码全是 Bug，好烦啊！"},
	}

	// 生成回复
	resp, err := provider.Generate(context.Background(), messages, &llm.GenerateOptions{Temperature: 0.7})
	if err != nil {
		panic(err)
	}

	fmt.Println("AI:", resp.Content)
}
```

## 📄 许可证
MIT License
