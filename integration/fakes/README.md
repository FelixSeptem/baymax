# integration/fakes 使用说明

## 功能域

`integration/fakes` 提供跨 integration 套件复用的测试桩，减少重复构造 model/tool/mcp 假实现。

## 关键入口

- `fakes.go`

## 使用约束

- 仅用于测试，不得被生产路径导入。
- 保持行为可预测（deterministic），避免引入时间/随机依赖导致 flaky。
- 若需要能力扩展，优先在现有 fake 上增量扩展，不复制新的并行 fake 实现。

## 验证命令

- `go test ./integration/... -count=1`
