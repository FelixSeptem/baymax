# Quickstart：真实 Provider

## 目的

这个示例在最小 runner 装配上接入 `model/openai`，用于验证宿主如何提供
Provider credential 和模型选择。它会执行真实网络请求，和离线的
`examples/01-chat-minimal` 不同。

## 前置条件

需要可用的 `OPENAI_API_KEY`。credential 只通过环境变量进入示例，不写入代码、
日志、diagnostics 或 replay fixture。

## 运行

```bash
export OPENAI_API_KEY="..."
go run ./examples/quickstart
```

PowerShell：

```powershell
$env:OPENAI_API_KEY = "..."
go run ./examples/quickstart
```

输出由 Provider 返回，内容和 token 使用量可能变化；成功标准是程序收到
`runner.Run` 的正常结果并打印 `FinalAnswer`。

## 该示例没有证明什么

- 它不证明业务授权、业务 effect、exactly-once 或持久化恢复。
- 它不自动发现 Provider，也不验证生产环境的 credential rotation、timeout、
  retry 或 rate limit policy。
- 它只走 `Run`；需要 Stream 时，使用同一 model/policy/tool 配置并验证
  [Run/Stream parity](../../docs/guides/integration.md)。

源码：[main.go](main.go)。Provider 协议边界见[Provider 组件说明](../../docs/components/providers.md)。
