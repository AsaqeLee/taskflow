# TaskFlow 内网发布检查清单

面向“这版是否可以作为内网 MVP / 试点版本发布”的最终收口清单。

- 具体部署步骤看 `INTRANET_RUNBOOK.md`
- 脚本说明看 `ACCEPTANCE_TESTING.md`
- 日常运维看 `INTRANET_OPS.md`

## 1. 先记录本次候选信息

每次准备试点或上线前，先补齐这四项：

- 候选日期
- 候选 commit / tag
- 执行人
- 对应校验记录路径，例如 `reports/intranet-release-assessment-YYYY-MM-DD.md`

说明：

- `reports/` 里的历史校验记录用于审计，不等于“当前仓库 HEAD 自动已通过发布校验”。
- 只有在本次候选版本重新完成校验后，才能把这份清单视为已执行。

## 2. 仓库内发布门槛

以下事项应在候选版本上逐项执行并留痕：

- [ ] 已确认发布范围仍为 MVP：不含用户管理 UI、自助密码重置页、公开注册页、任务分页搜索
- [ ] `DEV_MODE=false` 的严格生产参数校验已通过
- [ ] `STRICT_PRODUCTION_CONFIG=true` 已启用并通过正向校验
- [ ] `JWT_SECRET` / `APP_VERSION` / `BOOTSTRAP_USERS_FILE` / `CORS_ALLOWED_ORIGINS` 已能通过 `.env` 或 shell 注入
- [ ] 已确认受控发布入口可用：`bash scripts/intranet_release.sh .env`
- [ ] `go test ./...` 通过
- [ ] `cd web && npm run lint && npm run test && npm run build` 通过
- [ ] `bash scripts/compose_smoke.sh` 通过
- [ ] `bash scripts/nginx_smoke.sh` 通过
- [ ] `bash scripts/monitoring_smoke.sh` 通过
- [ ] `bash scripts/intranet_acceptance.sh` 通过
- [ ] 必要时补跑 `bash scripts/web_acceptance_smoke.sh`

## 3. 目标环境发布门槛

以下事项不在仓库内自动完成，但未满足前不能标记为“可试点部署”或“正式上线”：

- [ ] 已准备目标环境真实 `.env`
- [ ] 已生成并妥善保管真实 `JWT_SECRET`
- [ ] 已准备真实 `BOOTSTRAP_USERS_FILE`，并替换默认密码
- [ ] 已落实真实入口域名、HTTPS、反向代理和访问边界
- [ ] 已按 `INTRANET_RUNBOOK.md` 在目标环境完成部署
- [ ] `/readyz` 为 `200`
- [ ] 登录、建任务、分配、提交、审批、关闭主流程已在目标环境点验
- [ ] 同域入口或前后端分离入口已验证
- [ ] 已完成一次真实备份与恢复演练
- [ ] 已确认回滚入口、回滚命令和回滚责任人

## 4. Go / No-Go 判定

仅满足“仓库内发布门槛”时：

- 只能标记为“仓库内试点候选版”
- 不能标记为“已上线”或“企业生产级”

同时满足“仓库内发布门槛”和“目标环境发布门槛”时：

- 才可标记为“可试点部署”
- 再由业务 / 运维共同做 go / no-go 判定

## 5. 本次发布应产出的记录

- 一份候选校验记录，落在 `reports/`
- 一份目标环境部署 / 验收记录
- 一份备份恢复演练记录
- 一份最终 go / no-go 结论

## 6. 当前范围提醒

本阶段仍不包含：

- 用户管理 UI
- 自助密码重置 UI
- 公开注册 UI
- 分页搜索、附件、通知
- 企业级 SSO / MFA / 多租户
