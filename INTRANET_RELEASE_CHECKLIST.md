# TaskFlow 内网发布检查清单

面向“这版是否可以作为内网 MVP 发布”的收口清单。具体部署步骤看 `INTRANET_RUNBOOK.md`，脚本细节看 `ACCEPTANCE_TESTING.md`，日常运维看 `INTRANET_OPS.md`。

## 1. 发布门槛

以下项目全部满足，才视为本次版本可发布：

- [x] 已确认发布范围仍为 MVP：不含用户管理 UI、密码重置页、公开注册页、任务分页搜索
- [x] `.env` 或部署环境已设置 `DEV_MODE=false`（compose 默认 `false`，本地验收已验证）
- [ ] `STRICT_PRODUCTION_CONFIG=true` 已启用（禁止 `compose-local` / placeholder 生产参数）
- [ ] `JWT_SECRET` 已替换为强随机值（compose 示例值仅用于本地/CI；**生产部署前必须替换**）
- [~] `APP_VERSION` 已设置为明确版本号或 commit tag（本地 compose 为 `compose-local`；生产建议设为 git tag 或 commit short SHA）
- [ ] `BOOTSTRAP_USERS_FILE` 已指向自定义用户文件，且不再使用 `scripts/users.example.json` 默认密码
- [ ] `CORS_ALLOWED_ORIGINS` 已替换为实际内网前端域名（同域部署可与入口域名一致）
- [x] 本地后端验证通过：`go test ./...`（2026-06-22）
- [x] 本地前端验证通过：`cd web && npm run lint && npm run test && npm run build`（2026-06-22，17 tests）
- [x] Compose smoke 通过：`bash scripts/compose_smoke.sh`（2026-06-22）
- [x] 完整冷启动验收通过：`COLD_START=1 bash scripts/intranet_acceptance.sh`（2026-06-22）
- [x] 至少完成一次浏览器点验：owner + assignee 全流程（2026-06-22，`npm run acceptance:browser` Playwright 通过）
- [x] 已保留本次验收产物：日志、备份文件路径、版本号、执行人（见 §5）

## 2. 发布前检查

```bash
bash scripts/release_candidate_check.sh .env
bash scripts/validate_production_env.sh .env
go test ./...
cd web && npm run lint && npm run test && npm run build
bash scripts/compose_smoke.sh
COLD_START=1 bash scripts/intranet_acceptance.sh
```

需要同时确认：

- [x] `GET /readyz` 返回 `200`
- [x] owner 可 `GET /users?active=true`
- [x] 匿名 `POST /users` 返回 `401`
- [x] 备份恢复后任务仍可读
- [ ] `npm --prefix web run acceptance:responsive` 通过，或完成同等移动端人工点验

## 3. 部署顺序

1. 准备 `.env`，确认 `STRICT_PRODUCTION_CONFIG`、`JWT_SECRET`、`APP_VERSION`、`BOOTSTRAP_USERS_FILE`、`CORS_ALLOWED_ORIGINS`
2. 执行 `bash scripts/validate_production_env.sh .env`
3. 执行 `docker compose up -d --build`
4. 检查 `docker compose ps` 与 `curl -sf http://127.0.0.1:8080/readyz`
5. 用 bootstrap owner 账号执行一次登录验证
6. 按需启动前端演示：`cd web && npm run dev`，或托管 `web/dist`

## 4. 发布后人工点验

- [x] owner 登录成功（Playwright 2026-06-22）
- [x] assignee 登录成功（Playwright 2026-06-22）
- [x] owner 可创建任务（Playwright 2026-06-22）
- [x] owner 可分配任务（Playwright 2026-06-22）
- [x] assignee 可 `start`（Playwright 2026-06-22）
- [x] assignee 可 `submit`（Playwright 2026-06-22）
- [x] owner 可 `approve`（Playwright 2026-06-22）
- [x] owner 可 `close`（Playwright 2026-06-22）
- [x] 详情页可查看 records 与 audit logs（Playwright 2026-06-22）
- [x] 服务重启后任务数据仍在（`intranet_acceptance.sh` P3-11，2026-06-22）
- [ ] bootstrap 账号使用的已是自定义密码，而非 `change-me-*`

## 5. 发布记录

| 字段 | 填写 |
|------|------|
| 日期 | 2026-06-22 |
| 执行人 | 本地自动化验收（Agent） |
| Git commit | 待本轮生产加固提交后回填（当前基线：`0d7edf3c0cbf7f835da4fa6a2338b61521e29049`） |
| APP_VERSION | `compose-local`（本地验证值；生产部署时改为 tag/SHA） |
| 镜像 tag | `taskflow-taskflow:latest`（本地 compose build） |
| 验收结果 | **通过** — go test、前端 lint/test/build、compose smoke、冷启动 acceptance、Playwright 浏览器全流程 |
| 备份文件路径 | `backups/acceptance/acceptance-20260622-191906.gz` |
| 校验记录 | `reports/intranet-release-assessment-2026-06-22.md` |
| 已知风险 | compose 默认值仅用于本地/CI；生产仍需替换 `JWT_SECRET`、`APP_VERSION`、`BOOTSTRAP_USERS_FILE`、`CORS_ALLOWED_ORIGINS` 并启用 `STRICT_PRODUCTION_CONFIG=true`；移动端未人工点验；GitHub Actions 最新 run 已通过（2026-06-22） |

## 6. 当前已知非目标

- 用户管理后台
- 自助密码重置 UI
- 公开注册 UI
- 分页、搜索、附件、通知
- Agent 子任务建议、结构化反馈 schema、多人协作模型
