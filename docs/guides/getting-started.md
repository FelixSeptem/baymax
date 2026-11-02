# 开始使用 Baymax

这是一条面向外部开发者的最短接入路径。它先验证一个不依赖网络的
`Run`，再切换到真实 Provider，最后把宿主必须拥有的授权、持久化和业务
提交边界列出来。完整概念模型见[Execution Model](../architecture/execution-model.md)，
能力选择见[集成指南](integration.md)。

## 前置条件

- Go 1.26；仓库通过 `go.mod` 的 `toolchain` 声明验证工具链。建议使用当前
  Go 1.26 安全补丁版本，并用 `go version` 确认实际版本。
- 从仓库根目录运行命令；首次构建可能需要下载 Go module 依赖。
- 离线路径不需要 Provider credential。真实 Provider 路径需要由宿主通过
  环境变量或自己的 secret manager 提供 credential；Baymax 不保存或自动发现
  credential。

```bash
go version
go env GOTOOLCHAIN
```

## 五分钟离线路径

先运行 fake model 示例，确认 module、runner 和本地 Go 环境可用：

```bash
go run ./examples/01-chat-minimal
```

预期输出：

```text
hello from 01-chat-minimal
```

这个例子只验证最小 `runner.Run` 链路，不代表真实 Provider、工具、MCP、
重试、持久化或生产安全策略。对应的 source 和 README 是该路径的可执行
事实源：[examples/01-chat-minimal](../../examples/01-chat-minimal)。

## 第一次嵌入

Baymax 是 library-first runtime；宿主创建 model client、选择 policy，再显式
调用 runner。最小 model 只需要实现 `types.ModelClient` 的 `Generate` 和
`Stream`：

```go
engine := runner.New(modelClient)

run, err := engine.Run(ctx, types.RunRequest{Input: "say hi"}, nil)
if err != nil {
    return err
}

stream, err := engine.Stream(ctx, types.RunRequest{Input: "say hi"}, nil)
if err != nil {
    return err
}
_ = run
_ = stream
```

`Run` 和 `Stream` 共享 decision、tool、错误分类和 terminal 语义；Stream 只
改变事件交付方式。实际接入时应复用同一 policy、tool registry、recorder 和
宿主授权，不要为 Stream 建立平行的终态或业务提交路径。可编译的完整 fake
model 实现见 [minimal example](../../examples/01-chat-minimal/main.go)。

## 第一次真实 Provider 调用

仓库提供一个 OpenAI quickstart，展示 provider adapter 在
`model/openai`、credential 由宿主管理、runner 仍保持 SDK-neutral：

```bash
export OPENAI_API_KEY="..."
go run ./examples/quickstart
```

PowerShell：

```powershell
$env:OPENAI_API_KEY = "..."
go run ./examples/quickstart
```

该路径会产生真实网络调用和费用；不要把 key 写入源码、fixture、diagnostics
或 replay bundle。完整 source 见 [examples/quickstart](../../examples/quickstart)，
Provider 选择和 capability 约束见[Provider 组件说明](../components/providers.md)。
Baymax 不自动探测 Provider，也不从 package 名称推断 capability。

## 按需求继续

| 目标 | 下一步 | 先确认的边界 |
| --- | --- | --- |
| 本地 tool loop | [02-tool-loop-basic](../../examples/02-tool-loop-basic) | tool attempt/completion 属 tool owner；是否 retry 由 policy 决定 |
| MCP | [03-mcp-mixed-call](../../examples/03-mcp-mixed-call) | transport/session 属 MCP owner；敏感 header 必须脱敏 |
| Streaming/cancel | [04-streaming-interrupt](../../examples/04-streaming-interrupt) | cancellation request 不等于 source terminal cancellation |
| 多 agent / workflow | [集成指南](integration.md) | 先看 capability manifest；组合 continuation 不是 package 存在就自动可用 |
| 真实 source events | [native execution events](../../examples/native-execution-events) | callback 是实时 projection，不是 durable history/cursor |
| direct pause/resume | [host resume admission](host-resume-admission.md) | source checkpoint restore 与 host authorization 分离；业务 commit 仍由宿主 CAS |
| host plan/mailbox | [host execution plan](../../examples/host-execution-plan) | mailbox owner 保留 message/delivery/lease/retry/DLQ/terminal lifecycle |

## 接入前的宿主清单

在接入生产 Provider 或组合能力前，宿主应明确：

1. 当前 authority、tool allowlist、write scope、business revision 和 effect
   idempotency key 从哪里读取。
2. source history、checkpoint、lease、retry、DLQ 和 terminal 分别由哪个
   source owner 持有，是否真的支持 durable、replay 或 cross-process restore。
3. `RuntimeRecorder` 如何作为唯一 diagnostics writer 接收 bounded、脱敏的
   source projection。
4. source terminal 与 host business effect 如何通过独立 authorization/CAS/
   commit gate 关联；`ack`、`digest`、`RunResult` 或 observer status 都不能
   单独证明业务提交。
5. Run 和 Stream 是否使用相同的 policy、依赖 profile、错误断言和 rollback
   规则。

如果 capability、store、版本、identity 或 authority 不匹配，应在 owner
activation、publish、claim 或 effect 之前收到 typed error；不要通过重跑 parent、
从 latest cursor 开始或静默 direct fallback 来“恢复”。

## 验证与排查

先运行受影响 package 的 focused test，再扩大范围：

```bash
go test ./core/runner ./core/types ./integration -count=1
go test ./...
```

文档和 contract 变更还需要运行 [测试/replay/gate 指南](testing-replay-gates.md)
中的 replay、OpenSpec 和 docs consistency 检查。遇到文件占用、工具链下载或
严格漏洞扫描问题，按[故障排查](troubleshooting.md)记录实际退出码、风险、回滚点
和未执行项；不要把环境失败写成通过。

## 回滚

停用新增的 adapter、observer、plan binding 或 continuation profile，回到既有
source owner API；保留 source task/attempt/checkpoint/outbox/effect 事实，交给
宿主按原有幂等规则对账。回滚不能撤销已经发生的外部业务写入，也不能把历史
replay 当成恢复执行。
