# TaskFlow 补强计划执行证据（2026-06-17）

对照：`docs/plans/2026-06-17_16-32-11-taskflow-reinforcement-plan.md`

## 里程碑完成度

| 里程碑 | 状态 | 证据 |
|--------|------|------|
| M1 上线质量补强 | **完成** | 本节「本地验证」+ CI 配置 |
| M2 前端可维护性 | **完成** | TaskDetail 拆分 + `useAsyncData` + 13 前端单测 |
| M3 权限与运维 | **完成（V0 澄清）** | 权限矩阵 + runbook + 本节备份恢复 |
| M4 Agent 差异化 | **未开始（仅定义）** | `reports/agent-semantics-v0.md` |

## 本地验证（2026-06-17）

```text
go test ./... -count=1          → PASS
go test -race ./... -count=1    → PASS（同会话早期执行）
cd web && npm run lint          → PASS
cd web && npm run test          → PASS（4 files / 13 tests）
cd web && npm run build         → PASS
./scripts/intranet_acceptance.sh → ALL PASSED
```

验收脚本本次覆盖：

- bootstrap 用户登录
- 匿名 `POST /users` → 401
- owner `GET /users?active=true` 含 assignee
- 全流程：assign → start → submit → **reject → start → submit** → approve → close
- taskflow restart 后任务仍在
- mongodump → 删任务 → mongorestore --drop → 任务恢复

备份文件示例：`backups/acceptance/acceptance-20260617-*.gz`

## 备份恢复演练记录

| 字段 | 值 |
|------|-----|
| 日期 | 2026-06-17 |
| 执行人 | 本地自动化（intranet_acceptance.sh） |
| 环境 | docker compose / Colima |
| 结论 | **通过** |

## 已知未闭环（提交后仍需）

| 项 | 说明 |
|----|------|
| GitHub Actions | 本次仅本地验证，需 push 后看远端 CI |
| 浏览器 UI | P3-13 双人 UI 仍建议人工按 `reports/mobile-acceptance-checklist.md` 点验 |
| 权限收紧 | V0 保持 records/audit 任意 authenticated 可读，未改策略 |
| Agent 功能 | 仅语义文档，无结构化 TaskRecord 字段 |
| 本机 Mongo 集成测试 | host 连 `27018` replica set 可能因 `mongo:27017` 广播失败；以 CI 为准 |

## 关联交付物

- `ACCEPTANCE_TESTING.md`、`INTRANET_RUNBOOK.md`
- `reports/permission-matrix-v0.md`、`reports/mobile-acceptance-checklist.md`
- `.github/workflows/ci.yml`（frontend job + web_build_smoke + intranet_acceptance）