## TaskFlow 内网发布检查清单

面向“这版是否可以作为内网候选版本并推进正式发布”的最终收口清单。
具体发布步骤看 `INTRANET_RUNBOOK.md`，脚本说明看 `ACCEPTANCE_TESTING.md`，日常运维看 `INTRANET_OPS.md`。

## 1. 记录本次候选信息

- [ ] 候选日期已记录
- [ ] 候选 commit / tag 已记录
- [ ] 执行人已记录
- [ ] 对应校验记录路径已记录，例如 `reports/intranet-release-assessment-YYYY-MM-DD.md`

说明：

- `reports/` 中的历史记录仅用于审计，不代表当前版本自动通过。
- 每次新候选都要重新执行门禁并重新留痕。

## 2. 候选门禁必须通过

在候选环境或本地可复现实验环境执行：

```bash
bash scripts/release_candidate_check.sh .env
```

必须确认以下事项全部成立：

- [ ] `DEV_MODE=false` 的严格生产参数校验通过
- [ ] Go 全量测试通过
- [ ] 前端 `lint/test/build` 通过
- [ ] `scripts/security_audit.sh` 通过
- [ ] `scripts/compose_smoke.sh` 通过
- [ ] `scripts/intranet_acceptance.sh` 增量模式通过
- [ ] `scripts/intranet_acceptance.sh` 冷启动模式通过
- [ ] Hermes API key 生命周期通过：创建、认证、执行任务流、吊销后 401
- [ ] 备份恢复通过：`mongodump -> delete -> mongorestore` 后任务恢复
- [ ] `scripts/nginx_smoke.sh` 通过
- [ ] `scripts/web_acceptance_smoke.sh` 通过
- [ ] `scripts/monitoring_smoke.sh` 通过
- [ ] 候选门禁日志或 evidence 已保存

## 3. 正式发布前确认

- [ ] Docker / Docker Compose 在目标主机可用
- [ ] `APP_VERSION` 已固定为本次候选 commit / tag
- [ ] `JWT_SECRET`、`BOOTSTRAP_USERS_FILE`、Mongo 连接、密码重置 webhook 等生产参数已确认
- [ ] 已明确是否包含 same-origin web 入口
- [ ] 已明确回滚目标镜像或已设置 `TASKFLOW_PREVIOUS_IMAGE`
- [ ] 已确认本次发布只走 `bash scripts/intranet_release.sh .env`

## 4. 正式发布命令

API-only：

```bash
bash scripts/intranet_release.sh .env
```

包含 same-origin web：

```bash
TASKFLOW_RELEASE_INCLUDE_WEB=true bash scripts/intranet_release.sh .env
```

如果需要同时跑浏览器验收：

```bash
TASKFLOW_RELEASE_INCLUDE_WEB=true \
TASKFLOW_RELEASE_RUN_WEB_ACCEPTANCE=true \
bash scripts/intranet_release.sh .env
```

## 5. 发布后确认

- [ ] `/readyz` 返回 200
- [ ] `/metrics` 可访问
- [ ] API 登录、建任务、流转正常
- [ ] same-origin web 场景下首页和 `/api` 代理正常
- [ ] 关键日志、请求 ID、trace ID 可定位
- [ ] 如发布失败，优先执行 `scripts/rollback_image.sh`

## 6. 不要在正式环境临时替代的操作

- [ ] 不跳过候选门禁直接发布
- [ ] 不用 `X-User-ID`、legacy token 或手工伪造请求作为发布依据
- [ ] 不把 `scripts/intranet_acceptance.sh` 的备份恢复步骤直接当成线上临时巡检动作重复执行
- [ ] 不绕过 `scripts/intranet_release.sh` 手工拼接 compose / migrate / web 发布命令
