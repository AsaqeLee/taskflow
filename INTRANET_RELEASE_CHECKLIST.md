# TaskFlow 内网发布检查清单

面向“这版是否可以作为内网 MVP 发布”的收口清单。具体部署步骤看 `INTRANET_RUNBOOK.md`，脚本细节看 `ACCEPTANCE_TESTING.md`，日常运维看 `INTRANET_OPS.md`。

## 1. 发布门槛

以下项目全部满足，才视为本次版本可发布：

- [x] 已确认发布范围仍为 MVP：不含用户管理 UI、密码重置页、公开注册页、任务分页搜索
- [x] `.env` 或部署环境已设置 `DEV_MODE=false`
- [x] `STRICT_PRODUCTION_CONFIG=true` 已启用（本地 pilot `.env` 已验证）
- [x] `JWT_SECRET` 已替换为强随机值（`bash scripts/init_pilot_env.sh` 生成，不入 git）
- [x] `APP_VERSION` 已设置为明确版本号或 tag（pilot 使用 short SHA）
- [x] `BOOTSTRAP_USERS_FILE` 已指向自定义用户文件，且不再使用 `scripts/users.example.json` 默认密码
- [x] `CORS_ALLOWED_ORIGINS` 已按入口方式配置（同域 nginx / preview 可为空；分域时填真实域名）
- [x] 本地后端验证通过：`go test ./...`
- [x] 本地前端验证通过：`cd web && npm run lint && npm run test && npm run build`
- [x] 发布治理验证通过：`SKIP_GO_TEST=true bash scripts/security_audit.sh`
- [x] 至少完成一次浏览器点验：owner + assignee 全流程（Playwright）
- [x] 同域 nginx 入口通过：`bash scripts/nginx_smoke.sh`
- [x] 监控基线通过：`bash scripts/monitoring_smoke.sh`

## 2. 发布前命令

```bash
bash scripts/init_pilot_env.sh          # 或手工准备 .env
bash scripts/validate_production_env.sh .env
bash scripts/release_candidate_check.sh .env
bash scripts/backup_healthcheck.sh
```

需要同时确认：

- [x] `GET /readyz` 返回 `200`
- [x] owner 可 `GET /users?active=true`
- [x] 匿名 `POST /users` 返回 `401`
- [x] 备份恢复后任务仍可读
- [x] `scripts/backup_healthcheck.sh` 通过
- [x] Prometheus / Alertmanager 就绪并成功加载 TaskFlow 规则
- [x] 同域 nginx 入口可用且返回安全响应头
- [x] `acceptance:responsive` 通过（含在 `web_acceptance_smoke.sh`）

## 3. 部署顺序

1. `bash scripts/init_pilot_env.sh` 或手工准备 `.env`
2. `bash scripts/validate_production_env.sh .env`
3. `bash scripts/web_build_smoke.sh`（构建 `web/dist`）
4. `docker compose --env-file .env up -d --build`
5. 前端入口二选一：
   - **推荐试点：** `npm --prefix web run preview -- --host 0.0.0.0 --port 8081`
   - **生产 nginx：** `docker compose --env-file .env --profile full up -d`
6. `docker compose --env-file .env --profile monitoring up -d`
7. 检查 `curl -sf http://127.0.0.1:8080/readyz`
8. 检查 `curl -sf http://127.0.0.1:9090/-/ready` 与 `curl -sf http://127.0.0.1:9093/-/ready`
9. 用 `evidence/pilot-credentials.txt` 中的 owner 账号登录验证

## 4. 发布后点验

- [x] owner 登录成功（Playwright）
- [x] assignee 登录成功（Playwright）
- [x] owner 可创建任务（Playwright）
- [x] owner 可分配任务（Playwright）
- [x] assignee 可 `start`（Playwright）
- [x] assignee 可 `submit`（Playwright）
- [x] owner 可 `approve`（Playwright）
- [x] owner 可 `close`（Playwright）
- [x] 详情页可查看 records 与 audit logs（Playwright）
- [x] 服务重启后任务数据仍在（`intranet_acceptance.sh` P3-11）
- [x] bootstrap 账号使用自定义密码（pilot env 已验证）
- [x] 375px / 768px 核心页面可用（`acceptance:responsive`）

## 5. 发布记录

| 字段 | 填写 |
|------|------|
| 日期 | 2026-06-22 |
| 执行人 | 本地自动化 + pilot env 验收 |
| Git commit | 待本轮 P0 仓库加固提交后回填（当前基线：`57ebba8`） |
| APP_VERSION | pilot `.env` 使用当前 `git rev-parse --short HEAD` |
| 镜像 tag | `taskflow-taskflow:latest` |
| 验收结果 | **本地试点通过** — `release_candidate_check.sh .env` 全链路 |
| 备份文件路径 | 运行后回填最新 `backups/*.gz` |
| 校验记录 | `reports/intranet-release-assessment-2026-06-22.md` |
| 账号凭证 | `evidence/pilot-credentials.txt`（gitignored，勿提交） |
| 已知风险 | 目标内网主机尚未实配；当前仍是单机 compose 基线 |

## 6. 当前已知非目标

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- Agent 子任务建议、结构化反馈 schema、多人协作模型
