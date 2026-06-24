# TaskFlow 内网发布检查清单

面向“这版是否可以作为内网 MVP 发布”的最终收口清单。具体部署步骤看 `INTRANET_RUNBOOK.md`，脚本细节看 `ACCEPTANCE_TESTING.md`，日常运维看 `INTRANET_OPS.md`。

## 1. 当前候选版本

- 日期：2026-06-23
- 候选 commit：`dee3a20`
- 当前结论：仓库内发布门禁已通过，代码侧达到“可试点候选版”；目标内网环境尚未实际部署，暂不能标记为正式上线。
- 对应校验记录：`reports/intranet-release-assessment-2026-06-23.md`

## 2. 仓库内发布门槛

以下项目已在当前候选版本完成并验证：

- [x] 已确认发布范围仍为 MVP：不含用户管理 UI、密码重置页、公开注册页、任务分页搜索
- [x] `DEV_MODE=false` 的严格生产参数校验已通过
- [x] `STRICT_PRODUCTION_CONFIG=true` 已启用并通过正向校验
- [x] `JWT_SECRET` / `APP_VERSION` / `BOOTSTRAP_USERS_FILE` / `CORS_ALLOWED_ORIGINS` 已能通过 `.env` 或 shell 注入
- [x] `go test ./...` 通过
- [x] `cd web && npm run lint && npm run test && npm run build` 通过
- [x] `bash scripts/security_audit.sh` 通过
- [x] `bash scripts/compose_smoke.sh` 通过
- [x] `bash scripts/monitoring_smoke.sh` 通过
- [x] `bash scripts/nginx_smoke.sh` 通过
- [x] `bash scripts/pilot_smoke.sh .env` 通过
- [x] `bash scripts/web_acceptance_smoke.sh` 通过
- [x] `COLD_START=1 bash scripts/intranet_acceptance.sh` 通过
- [x] 严格发布门禁 `bash scripts/release_candidate_check.sh <tmp strict env>` 已在 `APP_VERSION=dee3a20` 下通过

## 3. 目标环境上线前必须补齐

以下项目不在仓库内自动完成，属于上线前的外部执行项：

- [ ] 目标内网主机、访问边界与运维账号已准备好
- [ ] 真实域名、HTTPS 证书与反向代理方案已确定
- [ ] 真实 `.env` 已生成，不再使用本地示例值
- [ ] 真实 bootstrap 用户文件已生成，并替换所有默认密码
- [ ] 目标环境 Mongo 副本集或等效生产拓扑已就绪
- [ ] Alertmanager 已配置真实通知去向
- [ ] 目标环境已生成首份正式备份，并登记保留路径
- [ ] 业务 owner / assignee 已完成一次真实试点签收

## 4. 目标环境 Go / No-Go 标准

满足以下条件，才建议标记为“可试点部署 / 可上线”：

- [ ] 按 `INTRANET_RUNBOOK.md` 在目标环境完成部署
- [ ] `curl -sf http://127.0.0.1:8080/readyz` 返回 `200`
- [ ] owner 能登录并查看用户列表
- [ ] owner + assignee 能走通创建、分配、启动、提交、驳回、重启、审批、关闭主流程
- [ ] 同域或 HTTPS 正式入口可正常访问，且安全头符合预期
- [ ] 目标环境已完成一次备份创建和一次恢复演练
- [ ] Prometheus / Alertmanager 在目标环境可用，且已接通真实告警通知
- [ ] 回滚镜像和回滚步骤已准备完毕

若以下任一项缺失，则建议保持“候选发布版”而非“上线版”：

- [ ] 仍使用示例密钥、示例密码或示例用户文件
- [ ] 仍未确认入口域名、HTTPS 或访问边界
- [ ] 仍未在目标环境执行完整验收
- [ ] 仍未完成目标环境备份与恢复验证

## 5. 发布记录

| 字段 | 当前事实 |
| --- | --- |
| 日期 | 2026-06-23 |
| 执行人 | 仓库内自动化 / 本地 compose 严格门禁 |
| Git commit | `dee3a20` |
| APP_VERSION | 当前严格门禁使用 `dee3a20` |
| 镜像 tag | `taskflow-taskflow:latest`（本地 compose 基线） |
| 验收结果 | **仓库内严格门禁通过**：`scripts/release_candidate_check.sh` 全链路通过 |
| 备份验证 | 已通过临时目录备份与恢复校验；目标环境正式备份路径待上线时登记 |
| 校验记录 | `reports/intranet-release-assessment-2026-06-23.md` |
| 账号凭证 | `evidence/pilot-credentials.txt`（gitignored，本地保留） |
| 已知风险 | 目标内网主机、TLS、真实告警路由尚未在仓库外落地 |

## 6. 当前已知非目标

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- Agent 子任务建议、结构化反馈 schema、多人协作模型
