# TaskFlow 补强阶段执行证据（2026-06-17）

本记录只保留当时的阶段执行结果，不再依赖 `docs/plans/` 下的本地计划稿。

## 里程碑完成度

- M1 上线质量补强：完成。证据是本地验证结果和 CI 配置。
- M2 前端可维护性：完成。证据是 TaskDetail 拆分、`useAsyncData` 和前端单测补齐。
- M3 权限与运维：完成（V0 澄清）。证据是权限矩阵、runbook 和备份恢复演练。
- M4 Agent 差异化：未开始（仅定义）。现阶段只保留 `reports/agent-semantics-v0.md`。

## 本地验证

- Go 单测通过。
- Go race 校验通过。
- 前端 lint、test、build 通过。
- `intranet_acceptance.sh` 通过。

本次验收脚本覆盖：

- bootstrap 用户登录
- 匿名 `POST /users` 返回 401
- owner `GET /users?active=true` 包含 assignee
- 主流程从 assign 到 approve / close 完整可走通，并覆盖 reject 后再次提交
- 服务重启后任务仍保留
- 备份、删库、恢复后任务可找回

备份文件示例：`backups/acceptance/acceptance-20260617-*.gz`

## 备份恢复演练记录

- 日期：2026-06-17
- 执行人：本地自动化（`intranet_acceptance.sh`）
- 环境：docker compose / Colima
- 结论：通过

## 当时仍未闭环的项

- GitHub Actions：当次记录仅保留本地验证，远端 CI 需以 push 后结果为准。
- 浏览器 UI：P3-13 双人 UI 仍建议按 `reports/mobile-acceptance-checklist.md` 人工点验。
- 权限收紧：V0 保持 records / audit 对 authenticated 可读，未进一步收紧。
- Agent 功能：仅保留语义文档，无结构化 TaskRecord 字段。
- 本机 Mongo 集成测试：host 直连 `27018` replica set 可能受 `mongo:27017` 广播影响，结果以 CI 为准。

## 关联交付物

- `ACCEPTANCE_TESTING.md`
- `INTRANET_RUNBOOK.md`
- `reports/permission-matrix-v0.md`
- `reports/mobile-acceptance-checklist.md`
- `.github/workflows/ci.yml`
