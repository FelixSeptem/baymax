# integration/adaptercontractreplay 测试说明

## 功能域

`adaptercontractreplay` 负责 adapter 合同回放与兼容性轨道验证，确保 fixture 驱动结果在 run/stream 路径保持语义等价。

## 关键内容

- 多轨道 fixture 回放：`v1alpha1`、`sandbox.v1`
- parse/activation 错误分类与字段定位一致性
- mixed-track backward compatibility
- sandbox drift class 稳定映射（backend/profile/manifest/session）

## 关键入口

- `replay_test.go`
- `integration/testdata/adapter-contract-replay/*`

## 验证命令

- `go test ./integration/adaptercontractreplay -count=1`

## 维护约束

- 新增 profile 版本时，必须更新回放轨道白名单与 fixture 覆盖。
- 回放结果断言应保持 deterministic，避免依赖不稳定环境变量。
