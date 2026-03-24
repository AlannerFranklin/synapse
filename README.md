# Synapse 🧠

一个主打 **并行执行 (DAG)** 与 **深度记忆 (Memory)** 的轻量级 Go 语言 LLM Agent 框架。

## 🌟 为什么选择 Synapse？

市面上已经有了 LangChain、Eino 等优秀的框架，但它们往往为了兼容性而引入了非常沉重的抽象层。
Synapse 旨在提供一个**极简、原生、高性能**的替代方案：
- **Go 原生并发**：利用 Go 的 `goroutine` 和 `channel`，天生支持多 Agent 节点的并行调度（Fan-out / Fan-in）。
- **零冗余抽象**：不到 1000 行核心代码，看懂源码只需要半天，极度容易二次开发。
- **多模型兼容**：一套代码，完美兼容 OpenAI、DeepSeek、硅基流动以及本地的 Ollama 模型。

---

## 🎯 核心进阶特性：带“决策回放”的 Agent (Replayable Agent)

在普通的 Agent 框架中，大模型的思考过程往往是一个“黑盒”。Synapse 引入了**企业级可观测性**设计：

- **🗂️ 状态快照 (State Trace)**：基于 DAG 图引擎，在每个节点执行时，自动记录上下文状态、动作与思考链 (Chain of Thought)。
- **⏪ 决策回放 (Replay)**：支持按时间轴回放 Agent 的完整思考过程（“为什么当时这么选”、“依据来自哪里”）。
- **🔍 异常溯源**：当任务执行失败时，可精确追溯到具体失败的并发节点与历史状态。

*简历亮点建议写法：*
> **“设计可回放的 Agent Memory System，支持决策溯源、上下文快照恢复与证据链接。基于 Go 并发特性实现 DAG 并行图调度器，显著提升多步任务执行效率。”**

---

## 🚀 开发进度

### Phase 1: 基础架构 (已完成)
- [x] **统一 Schema**：标准化 Message, Tool, State 数据结构
- [x] **LLM Provider**：极简的 OpenAI 兼容客户端 (支持 DeepSeek)
- [x] **DAG 执行引擎**：基于图的串行/并行任务调度器
- [x] **Memory 模块**：Short-term 短期滑动窗口记忆 & Long-term 长期文件记忆

### 🔒 安全提醒：配置 API Key
为了防止 API Key 泄漏到 GitHub，本项目使用本地文件读取的方式：
1. 在项目根目录创建 `api.txt` 文件。
2. 将你的 DeepSeek API Key（例如 `sk-xxxxxx`）直接粘贴到文件中，不要包含任何多余空格或换行。
3. `api.txt` 已被加入 `.gitignore`，不会被提交到版本库中。

### Phase 2: 高级可观测性 (进行中)
- [x] **TraceLog 结构**：改造全局状态，支持执行轨迹记录
- [x] **JSON 思考链解析**：强迫 LLM 输出思考过程并捕获
- [ ] **回放引擎 (Replayer)**：在终端实现 `/trace` 命令回放功能

## 📄 许可证
MIT License
