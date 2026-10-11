# 原生执行事件接入

## Semantic anchors 与 runtime path

真实 runner.Engine.Run/Stream -> host/eventadapter.Observer -> 宿主内存 sink。fake model 使用 adapter/modelcapability 的显式适配，无网络、tool effect 或真实业务数据。native_execution_event.v1 保留 native type、source、Run、iteration、payload；mapping_reference 不能用于去重。

## Expected markers

运行 go run ./examples/native-execution-events，输出 native-events run=completed、native-events stream=completed、observer=healthy 和 pause-is-terminal=false。input_required 是暂停；confirm resolution 不表示 canceled。observer.Status 独立于业务 terminal，sink error 是 unconfirmed，必须由业务 commit gate 决定提交。

## 能力与回滚

direct callback 无 history/cursor/dedup/global sequence；Realtime 仅通过既有 binding 纯投影保留 source 能力。默认交付 deadline 1 秒，sink 必须响应 context；默认引用 1024 bytes/16 citations/8 KiB，event 64 KiB。停用 observer 恢复原 handler，保留宿主记录；示例不修改 agent-modes。
