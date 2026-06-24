# TaskFlow 内网发布校验记录（2026-06-23）

面向“这版是否已经具备内网试点候选资格”的正式、受版本控制校验记录。

## 1. 校验基线

- 校验日期：2026-06-23
- 校验对象：`dee3a20 feat: harden release gate and ops baseline`
- 校验环境：本地 macOS + Colima Docker
- 校验方式：自动化 + Compose + 浏览器主流程 + 严格发布门禁

## 2. 本次已执行验证

以下检查已在当前代码线上通过：

- `go test ./...`
- `cd web && npm run lint && npm run test && npm run build`
- `bash scripts/security_audit.sh`
- `bash scripts/compose_smoke.sh`
- `BACKUP_TOOL=compose bash scripts/backup_mongo.sh` + `bash scripts/backup_healthcheck.sh`
- `bash scripts/monitoring_smoke.sh`
- `bash scripts/nginx_smoke.sh`
- `bash scripts/web_acceptance_smoke.sh`
- `COLD_START=1 bash scripts/intranet_acceptance.sh`
- `bash scripts/release_candidate_check.sh <tmp strict env>`，其中 `APP_VERSION=dee3a20`

## 3. 关键结果

### 3.1 代码与自动化

- Go 测试通过
- 前端 lint / test / build 通过
- 安全审计通过，包括 `go vet`、`govulncheck`、`npm audit`
- Compose smoke 通过
- 冷启动 acceptance 通过
- Playwright 浏览器主流程和响应式验收通过

### 3.2 生产参数与发布门禁

当前仓库已具备以下生产参数接入能力：

- `docker-compose.yml` 可从 `.env` / shell 读取：
  - `JWT_SECRET`
  - `APP_VERSION`
  - `BOOTSTRAP_USERS_FILE`
  - `CORS_ALLOWED_ORIGINS`
- `scripts/validate_production_env.sh` 可阻止明显错误的生产参数：
  - `DEV_MODE=true`
  - `STRICT_PRODUCTION_CONFIG=false`
  - 占位符 `JWT_SECRET`
  - 占位符 `APP_VERSION`
  - `BOOTSTRAP_USERS_FILE` 仍指向 `scripts/users.example.json`
  - 本地开发 CORS origin
- `scripts/release_candidate_check.sh` 现已覆盖：
  - Go 测试
  - 前端 lint / test / build
  - 安全审计
  - Compose smoke
  - 备份校验
  - 监控校验
  - 浏览器验收
  - nginx 同域入口校验
  - 冷启动验收

### 3.3 备份、监控与入口

- 备份链路通过：归档、校验和、metadata、freshness check 均通过
- Prometheus / Alertmanager profile 通过：taskflow scrape 正常、告警规则成功加载
- nginx 同域入口通过：`/api` 代理和关键安全响应头验证通过
- Vite preview 生产包入口已单独验证通过，可继续作为手工演示辅助入口，但不再属于严格发布门禁

## 4. 执行中发现的问题

本轮校验中出现过 1 次环境问题，但不是代码缺陷：

- 首次重跑严格发布门禁时，本机 Colima 未启动，`docker` 无法连接到 `unix:///Users/asaqelee/.colima/default/docker.sock`
- 处理方式：执行 `colima start`
- 结果：恢复 Docker runtime 后，重新运行当前候选版本严格门禁并通过

## 5. 当前结论

结论：**`dee3a20` 已达到“内网试点候选版”标准。**

这代表：

- 仓库内代码、脚本、文档、发布门禁已经闭环
- 本地严格发布门禁可稳定跑通
- 继续推进项目的主阻塞点已经不在仓库内，而在目标环境落地

## 6. 仍未完成的外部执行项

以下项目仍需在目标内网环境完成，完成后才能给出正式 go / no-go：

- 真实主机部署
- 真实域名与 HTTPS
- 真实 `.env`、真实 bootstrap 用户文件、真实密钥
- 真实 Alertmanager 通知去向
- 目标环境正式备份路径登记与恢复演练
- 业务 owner / assignee 试点签收
