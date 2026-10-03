# Dynamic action resume example

语义锚点：`dynamic_action.input_required_same_run`。

运行路径：`core/runner -> core/types -> host`。工具只返回 opaque token、kind、correlation 和 digest；PendingAction 业务正文、授权、确认策略和 executor 均留在 adapter 侧，不进入 Baymax checkpoint。

预期标记：首次 Run 返回 `input_required`，且不会执行下一次 model step；确认后使用相同 `run_id` 从 checkpoint 继续并完成。重复使用同一 idempotency key 必须幂等。

```powershell
go run ./examples/dynamic-action-resume
```

回滚说明：移除 `PendingAction` registration 即恢复既有 tool-result -> next-model loop；Realtime resume、retry 和 follow-up promotion 不经过此路径。
