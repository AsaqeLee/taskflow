# Agent 角色语义（V0）

## 定位

`agent` 在本阶段是**可登录的执行者账号类型**，不是独立子系统。与 `human` 的差异主要体现在运维与产品叙事，而非额外 API。

## V0 行为定义

| 场景 | 行为 |
|------|------|
| 认证 | 与 human 相同：密码登录 + JWT |
| 创建任务 | **允许**。任何已登录用户均可创建，agent 不例外 |
| 被分配 | 可作为 assignee 接收任务 |
| 执行 | assignee 可 `start` / `submit` |
| 审批/分配 | 仅当该 agent 账号是任务 **creator** 时 |
| 用户管理 | 无；由 owner 通过 API/bootstrap 管理 |

## 与传统工单系统的差异（可对外说明）

1. 任务流明确区分 **creator（验收方）** 与 **assignee（执行方）**，agent 主要承担执行方。
2. 提交/驳回/审批均留下 **TaskRecord** 与 **AuditLog**，便于追溯 agent 产出。
3. 后续可在此基础上增加结构化反馈字段，而无需重构权限核心。

## 本期不做

- Agent 自动提交失败原因的结构化 schema（第二期）
- 子任务建议 / 拆分机制
- 多人协作、观察者、委派链
- Agent 专用 UI（与 human 共用 `web/`）

## 演进建议

当需要差异化时，优先：

1. 规范 `TaskRecord.content` 模板（摘要 / 阻塞 / 失败原因）。
2. 为 agent 任务增加可选 metadata（不破坏现有 API）。
3. 再评估子任务建议 API。