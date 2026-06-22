# TaskFlow 内网发布校验记录（2026-06-22）

面向“这版是否已经具备内网试点候选资格”的正式、受版本控制校验记录。

## 1. 校验基线

- 校验日期：2026-06-22
- 校验对象：基于 `0d7edf3c0cbf7f835da4fa6a2338b61521e29049` 的当前工作树（含本轮生产加固改动）
- 校验环境：本地
- 校验方式：自动化 + Compose + 浏览器主流程 + GitHub Actions

## 2. 已执行验证

以下检查已在当前代码线上通过：

- `go test ./...`
- `cd web && npm run lint && npm run test && npm run build`
- `bash scripts/compose_smoke.sh`
- `COLD_START=1 bash scripts/intranet_acceptance.sh`
- `cd web && npm run acceptance:browser`
- `gh run list --limit 5`

## 3. 关键结果

### 3.1 代码与自动化

- Go 测试通过
- 前端 lint / test / build 通过
- Compose smoke 通过
- 冷启动 acceptance 通过
- Playwright 浏览器主流程通过
- 最新 `main` push 的 GitHub Actions `ci` 通过

### 3.2 生产参数能力

当前仓库已具备以下生产参数接入能力：

- `docker-compose.yml` 可从 `.env` / shell 读取：
  - `JWT_SECRET`
  - `APP_VERSION`
  - `BOOTSTRAP_USERS_FILE`
  - `CORS_ALLOWED_ORIGINS`
- `.env.intranet.example` 已提供生产参数模板
- `STRICT_PRODUCTION_CONFIG=true` 可阻断明显的示例值 / 本地值误上生产
- `scripts/validate_production_env.sh` 可在部署前校验：
  - 强随机 `JWT_SECRET`
  - 真实 `APP_VERSION`
  - 自定义 `BOOTSTRAP_USERS_FILE`
  - bootstrap 文件中不含 `change-me-*`
  - 非 localhost 的 `CORS_ALLOWED_ORIGINS`

### 3.3 验收产物

- 最新冷启动验收备份：`backups/acceptance/acceptance-20260622-191906.gz`

## 4. 当前结论

### 4.1 已达到

- 项目已达到“候选发布版”状态
- 代码、自动化、Compose、浏览器主路径、CI 都已通过
- 生产参数接入与生产级配置护栏已具备

### 4.2 尚未完成

以下事项仍需在目标环境完成，之后才应标记为“可试点部署”：

- 用真实值填充目标 `.env`
- 启用 `STRICT_PRODUCTION_CONFIG=true`
- 使用自定义 bootstrap 用户文件并替换默认密码
- 确认真实入口域名 / 同域或分域方式
- 在目标内网环境完成一次真实部署与主流程点验
- 完成移动端 / 窄屏人工点验

## 5. 放行判断

当前判断：

- 对“代码是否健康、功能是否闭环、自动化是否通过”的答案：通过
- 对“是否可以直接按仓库默认 compose 值上内网”的答案：不通过
- 对“是否已经具备试点部署候选资格”的答案：通过
  - 前提：按 `INTRANET_RUNBOOK.md` 与 `scripts/validate_production_env.sh` 填入真实生产参数后再部署

## 6. 关联文档

- `INTRANET_RELEASE_CHECKLIST.md`
- `INTRANET_RUNBOOK.md`
- `INTRANET_OPS.md`
- `INTRANET_PILOT_RELEASE_PLAN.md`
- `reports/mobile-acceptance-checklist.md`
