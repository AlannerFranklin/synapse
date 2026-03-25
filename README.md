# Synapse - 模块化 AI Agent 框架

**Synapse** 是一个从零开始使用纯 Go 语言编写的模块化 AI Agent 执行框架。它探索并实现了从简单的函数调用，到有向无环图（DAG），再到基于蓝图树（Blueprint Tree）的复杂状态执行模型。本项目旨在提供一个轻量、透明且可扩展的 Agent 构建范式。

## 🌟 核心特性

本项目经历了三个阶段的演进，逐步实现了越来越强大的 Agent 能力：

### Phase 1: 基础架构与 DAG 引擎
- **Schema 定义**: 统一的消息（Message）、状态（State）和工具（Tool）数据结构。
- **大模型接入**: 原生支持 DeepSeek/OpenAI 兼容的 API 调用。
- **记忆管理**: 实现了基于滑动窗口的短期记忆（ShortTermMemory）。
- **DAG 执行引擎**: 支持串行和并行的节点执行图，能够将复杂任务拆解为多个子节点。

### Phase 2: 可观测性与持久化
- **全局状态追踪**: 引入 `TraceLog` 结构，详细记录每个节点的执行过程和状态流转。
- **JSON 思考链**: 强制大模型输出 JSON 格式的思考过程（Chain of Thought）并进行结构化解析。
- **终端回放**: 实现 `/trace` 命令，在终端随时查看 Agent 的内部思考和执行轨迹。
- **长期记忆**: 增加了简单的文件系统长期记忆持久化，实现基础的记忆固化功能。

### Phase 3: 蓝图树 (Blueprint Tree) 引擎 [当前进展]
- **双向多叉树结构**: 引入了带有 `Parent` 和 `Children` 指针的 `TreeNode`。
- **状态隔离**: 实现了全局状态的深拷贝（`State.Clone()`），保证每个节点拥有独立的执行上下文。
- **非递归遍历**: 实现基于队列（BFS/DFS）的非递归执行调度，防止深树导致栈溢出。
- **分支推演准备**: 核心基建已完成，即将支持“多分支推演”（Tree of Thoughts）和“时光回溯”功能。

## 🚀 快速开始

### 1. 配置 API Key
在项目根目录创建一个 `api.txt` 文件，填入你的大模型 API Key（目前默认使用 DeepSeek 接口，可自行在 `main.go` 中修改 BaseURL）。
> **注意**: `api.txt` 已被加入 `.gitignore`，请勿将你的真实 Key 提交到版本控制系统。

### 2. 运行项目
确保你已经安装了 Go (>= 1.20)，在终端执行：
```bash
go run main.go
```

### 3. 交互命令
- 输入普通文本与 AI 进行对话。
- 输入 `/trace` 查看刚刚执行的完整内部思考过程。
- 输入 `exit` 退出程序。

## 📂 目录结构

```text
synapse/
├── graph/       # 执行引擎核心 (包含 DAG 和 Blueprint Tree 实现)
├── llm/         # 大模型 API 客户端 (支持 OpenAI 兼容格式)
├── memory/      # 短期记忆与上下文管理
├── schema/      # 全局数据结构 (State, Message, TraceLog 等)
├── main.go      # 项目入口和装配逻辑
└── README.md    # 项目说明
```

## 📜 许可证

本项目采用 [Apache License 2.0](LICENSE) 开源协议。你可以自由地使用、修改和分发代码，但请保留原作者的版权声明。
