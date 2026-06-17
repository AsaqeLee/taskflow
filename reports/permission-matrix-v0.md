# TaskFlow V0 权限矩阵

与 `internal/domain/task/task.go`、`web/src/domain/taskWorkflow.ts`、handler 测试对齐。

## 任务生命周期

| 动作 | 状态前提 | 允许角色 | 后端错误码 |
|------|----------|----------|------------|
| 创建任务 | — | 任意已登录用户（含 `agent`） | — |
| 更新标题/描述 | 任意非删除 | **creator** | `forbidden` |
| 分配 assign | `open` | **creator** | `forbidden` / `invalid_request` |
| 开始 start | `assigned` | **assignee** | `forbidden` |
| 提交 submit | `in_progress` | **assignee** | `forbidden` |
| 驳回 reject | `submitted` | **creator** | `forbidden` |
| 审批 approve | `submitted` | **creator** | `forbidden` |
| 关闭 close | `approved` | **creator** | `forbidden` |
| 取消 cancel | `open`/`assigned`/`in_progress`/`submitted` | **creator** | `forbidden` |
| 重新激活 reactivate | `cancelled`/`completed` | **creator** | `forbidden` |
| 删除 delete | 无状态硬限制（V0） | **creator** | `forbidden` |

前端在 `open`/`assigned` 才展示 delete 按钮（比后端更严，可接受）。

## 用户与身份

| 动作 | 允许角色 | 备注 |
|------|----------|------|
| `GET /users` 全量 | `owner` | 其他人仅返回自己 |
| `GET /users?active=true` | `owner` | assign 下拉用 |
| `POST /users` | `owner`（鉴权下） | `ALLOW_PUBLIC_REGISTER=false` |
| 冷启动创建用户 | bootstrap 直连 DB | 无 token 时的唯一路径 |
| `POST /users/:id/disable` | `owner` 或本人 | — |

## 可读性（V0 有意放宽）

| 资源 | 当前策略 | 后续可收紧 |
|------|----------|------------|
| `GET /tasks` | 任意已登录 | 第二期可按参与方过滤 |
| `GET /tasks/:id` | 任意已登录 | 同上 |
| `GET /tasks/:id/records` | 任意已登录（知 task id） | creator/assignee/owner |
| `GET /tasks/:id/audit_logs` | 任意已登录 | 同上 |

测试：`internal/handler/task_access_test.go` 固化 records 的 V0 行为。

## Agent 角色（V0）

| 能力 | 是否允许 |
|------|----------|
| 登录 | 是 |
| 创建任务 | **是**（与 human 相同，无额外限制） |
| 被分配为 assignee | 是 |
| start / submit | 是（作为 assignee） |
| 分配他人 / 审批 | 仅当其同时为 creator |

详见 `reports/agent-semantics-v0.md`。